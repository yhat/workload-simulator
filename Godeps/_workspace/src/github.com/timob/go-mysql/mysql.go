// mysql driver for database/sql
package mysql

// see http://dev.mysql.com/doc/internals/en/client-server-protocol.html
// for the protocol definition.

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type mysql struct{}

type conn struct {
	protocolVersion    byte
	serverVersion      string
	version            []byte
	connId             uint32
	serverCapabilities uint16
	serverLanguage     uint8
	serverStatus       uint16
	host               string
	port               int
	user               string
	password           *string
	db                 string
	netconn            net.Conn
	bufrd              *bufio.Reader
	tls                *tls.Config
	socket             string
	strict             bool
	debug              bool
	allowLocalInfile   bool
	charset            string
	seq                byte
	clientMultiResults bool
}

type stmt struct {
	cn       *conn
	qs       string
	stmtId   uint32
	params   []column
	columns  []column
	warnings uint16
}

type column struct {
	name     string
	charset  uint16
	length   uint32
	coltype  byte
	flags    uint16
	decimals byte
}

type result struct {
	cn           *conn
	columns      []column
	binary       bool
	closed       bool
	rowsAffected int64
	lastInsertId int64
	warnings     uint16
	status       uint16
}

func init() {
	sql.Register("mysql", &mysql{})
}

func (d *mysql) Open(name string) (driver.Conn, error) {
	conn, err := connect(name)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func connect(dsn string) (*conn, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid dsn: %s", dsn)
	}

	cn := &conn{host: "localhost", port: 3306, user: "root", socket: "/var/run/mysqld/mysqld.sock"}

	switch u.Scheme {
	case "mysql":
	case "mysqls":
		cn.tls = &tls.Config{}
	default:
		return nil, fmt.Errorf("invalid scheme: %s", dsn)
	}

	for k, v := range u.Query() {
		switch k {
		case "debug":
			cn.debug = true
		case "ssl-insecure-skip-verify":
			if cn.tls != nil {
				cn.tls.InsecureSkipVerify = true
			}
		case "allow-insecure-local-infile":
			cn.allowLocalInfile = true
		case "charset":
			cn.charset = v[0]
		case "socket":
			cn.socket = v[0]
		case "strict":
			cn.strict = true
		case "client-multi-results":
			cn.clientMultiResults = true
		default:
			return nil, fmt.Errorf("invalid parameter: %s", k)
		}
	}

	if len(u.Host) > 0 {
		host_port := strings.SplitN(u.Host, ":", 2)
		cn.host = host_port[0]

		if len(host_port) == 2 {
			cn.port, err = strconv.Atoi(host_port[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", dsn)
			}
		}
	}

	if u.User != nil {
		cn.user = u.User.Username()
		if p, ok := u.User.Password(); ok {
			cn.password = &p
		}
	}

	if len(u.Path) > 0 {
		path := strings.SplitN(u.Path, "/", 2)
		cn.db = path[1]
	}

	if u.Host == "(unix)" {
		cn.netconn, err = net.Dial("unix", cn.socket)
	} else {
		cn.netconn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", cn.host, cn.port))
	}
	if err != nil {
		return nil, err
	}
	if err = cn.hello(); err != nil {
		cn.netconn.Close()
		return nil, err
	}

	cn.bufrd = bufio.NewReader(cn.netconn)

	if cn.debug {
		log.Printf("connected: %s #%d (%s)\n", dsn, cn.connId, cn.serverVersion)
	}
	if cn.charset != "" {
		if _, err := cn.Exec("SET NAMES "+cn.charset, nil); err != nil {
			return nil, err
		}
	}
	return cn, nil
}

func (cn *conn) newComPacket(com byte) (p packet) {
	cn.seq = 0
	p = newPacket()
	p.WriteByte(com)
	return p
}

func (cn *conn) recvPacket() (p packet, err error) {
	cn.seq, err = p.recv(cn.bufrd, cn.seq)
	return p, err
}

func (cn *conn) sendPacket(p packet) (err error) {
	err = p.send(cn.netconn, cn.seq)
	cn.seq += 1
	return err
}

func (cn *conn) hello() error {
	challange, err := cn.readHello()
	if err != nil {
		return err
	}
	if cn.tls != nil {
		if cn.serverCapabilities&CLIENT_SSL == 0 {
			return fmt.Errorf("server does not support SSL")
		}
		if err = cn.writeHello(nil, CLIENT_SSL); err != nil {
			return err
		}
		cn.netconn = tls.Client(cn.netconn, cn.tls)
	}
	if err := cn.writeHello(challange, 0); err != nil {
		return err
	}

	var p packet
	if cn.seq, err = p.recv(cn.netconn, cn.seq); err != nil {
		return err
	}

	switch p.FirstByte() {
	case OK:
	case ERR:
		return p.ReadErr()
	default:
		return fmt.Errorf("hello: expected OK or ERR, got %v", p.FirstByte())
	}

	return nil
}

func (cn *conn) readHello() (challange []byte, err error) {
	var p packet
	if cn.seq, err = p.recv(cn.netconn, cn.seq); err != nil {
		return nil, err
	}
	cn.protocolVersion = p.ReadUint8()
	if s, err := p.ReadString('\x00'); err != nil {
		return nil, err
	} else {
		cn.serverVersion = s[:len(s)-1]
	}

	if cn.version, err = parseVersion(cn.serverVersion); err != nil {
		log.Printf("warning: could not parse server version '%s'\n", cn.serverVersion)
	}

	cn.connId = p.ReadUint32()
	challange = p.Next(8)
	p.Next(1)
	cn.serverCapabilities = p.ReadUint16()
	cn.serverLanguage = p.ReadUint8()
	cn.serverStatus = p.ReadUint16()
	p.Next(13)
	challange = append(challange, p.Next(12)...)
	p.Next(1)

	return challange, nil
}

func (cn *conn) writeHello(challange []byte, flags uint32) error {
	p := newPacket()
	flags |= CLIENT_PROTOCOL_41 | CLIENT_SECURE_CONNECTION | CLIENT_LOCAL_FILES
	if len(cn.db) > 0 {
		flags |= CLIENT_CONNECT_WITH_DB
	}
	if cn.clientMultiResults {
		flags |= CLIENT_MULTI_RESULTS | CLIENT_PS_MULTI_RESULTS
	}
	p.WriteUint32(flags)
	p.WriteUint32(MAX_PACKET_SIZE)
	if bytes.Compare(cn.version, []byte{5, 5, 3}) >= 0 {
		p.WriteByte(CHARSET_UTF8MB4)
	} else {
		p.WriteByte(CHARSET_UTF8)
	}
	p.Write(make([]byte, 23))

	if flags&CLIENT_SSL == 0 {
		p.WriteString(cn.user)
		p.WriteByte(0)
		if cn.password != nil {
			token := passwordToken(*cn.password, challange)
			p.WriteByte(byte(len(token)))
			p.Write(token)
		} else {
			p.WriteByte(0)
		}
		if len(cn.db) > 0 {
			p.WriteString(cn.db)
			p.WriteByte(0)
		}
	}
	err := cn.sendPacket(p)
	return err
}

func passwordToken(password string, challange []byte) (token []byte) {
	d := sha1.New()

	d.Write([]byte(password))
	h1 := d.Sum(nil)

	d.Reset()
	d.Write(h1)
	h2 := d.Sum(nil)

	d.Reset()
	d.Write(challange)
	d.Write(h2)
	token = d.Sum(nil)

	for i := range token {
		token[i] ^= h1[i]
	}

	return token
}

func parseVersion(versionString string) (version []byte, err error) {
	parts := strings.Split(versionString, "-")
	for _, s := range strings.Split(parts[0], ".") {
		v, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		version = append(version, byte(v))
	}
	return version, nil
}

func (cn *conn) Begin() (driver.Tx, error) {
	if _, err := cn.Exec("BEGIN", nil); err != nil {
		return nil, err
	}
	return cn, nil
}

func (cn *conn) Commit() error {
	_, err := cn.Exec("COMMIT", nil)
	return err
}

func (cn *conn) Rollback() error {
	_, err := cn.Exec("ROLLBACK", nil)
	return err
}

func (cn *conn) Close() (err error) {
	p := cn.newComPacket(COM_QUIT)
	if err := cn.sendPacket(p); err != nil {
		return err
	}
	if _, err = cn.recvPacket(); err != nil {
		return err
	}
	return cn.netconn.Close()
}

func (cn *conn) readColumns(n int) ([]column, error) {
	if n == 0 {
		return nil, nil
	}

	cols := make([]column, n)
	for i := range cols {
		p, err := cn.recvPacket()
		if err != nil {
			return nil, err
		}
		col := &cols[i]
		p.SkipLCBytes()                // catalog
		p.SkipLCBytes()                // schema
		p.SkipLCBytes()                // table
		p.SkipLCBytes()                // org_table
		col.name, _ = p.ReadLCString() // name
		p.SkipLCBytes()                // org_name
		p.ReadLCUint64()               // 0x0c
		col.charset = p.ReadUint16()
		col.length = p.ReadUint32()
		col.coltype = p.ReadUint8()
		col.flags = p.ReadUint16()
		col.decimals = p.ReadUint8()
	}
	p, err := cn.recvPacket()
	if err != nil {
		return nil, err
	}
	if x := p.ReadUint8(); x != EOF {
		return nil, fmt.Errorf("readColumns: expected EOF, got %v", x)
	}
	return cols, nil
}

func (cn *conn) Exec(query string, args []driver.Value) (driver.Result, error) {
	if len(args) > 0 {
		return nil, driver.ErrSkip // fall back to prepare/exec
	}
	if cn.debug {
		log.Println("exec:", query)
	}
	return cn.exec(query)
}

func (cn *conn) exec(query string) (r *result, err error) {
	if r, err = cn.query(query); err != nil {
		return nil, err
	}
	if err = r.Close(); err != nil {
		return nil, err
	}
	return r, nil
}

func (cn *conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	if len(args) > 0 {
		return nil, driver.ErrSkip // fall back to prepare/exec
	}
	if cn.debug {
		log.Println("query:", query)
	}
	return cn.query(query)
}

func (cn *conn) query(query string) (r *result, err error) {
	if len(query) > MAX_PACKET_SIZE {
		return nil, fmt.Errorf("query exceeds %d bytes", MAX_PACKET_SIZE)
	}
	p := cn.newComPacket(COM_QUERY)
	p.WriteString(query)
	if err = cn.sendPacket(p); err != nil {
		return nil, err
	}

	r = &result{cn: cn}

	if err = r.readResponse(); err != nil {
		return nil, err
	}

	return r, nil
}

func (cn *conn) Prepare(query string) (driver.Stmt, error) {
	if cn.debug {
		log.Printf("prepare: %s", query)
	}
	return cn.prepare(query)
}

func (cn *conn) prepare(query string) (st *stmt, err error) {
	p := cn.newComPacket(COM_STMT_PREPARE)
	p.WriteString(query)
	if err := cn.sendPacket(p); err != nil {
		return nil, err
	}
	if p, err = cn.recvPacket(); err != nil {
		return nil, err
	}

	st = &stmt{cn: cn, qs: query}
	switch p.FirstByte() {
	case OK:
		p.ReadUint8() // OK
		st.stmtId = p.ReadUint32()
		numColumns := int(p.ReadUint16())
		numParams := int(p.ReadUint16())
		p.ReadUint8() // filler
		if p.Len() >= 2 {
			// mysql 4.1 does not always supply this field
			st.warnings = p.ReadUint16()
		}
		st.cn.logWarnings(st.warnings)
		if st.params, err = cn.readColumns(numParams); err != nil {
			return nil, err
		}
		if st.columns, err = cn.readColumns(numColumns); err != nil {
			return nil, err
		}
	case ERR:
		return nil, p.ReadErr()
	default:
		return nil, fmt.Errorf("prepare: expected OK or ERR, got %v", p.FirstByte())
	}
	return st, nil
}

func (cn *conn) sendLocalFile(r *result, fn string) error {
	if !cn.allowLocalInfile {
		return fmt.Errorf("client does not allow LOAD DATA LOCAL")
	}
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, MAX_DATA_CHUNK)
	for {
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n > 0 {
			p := newPacket()
			p.Write(buf[:n])
			if err := cn.sendPacket(p); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
	}
	p := newPacket()
	if err = cn.sendPacket(p); err != nil {
		return err
	}

	if p, err = cn.recvPacket(); err != nil {
		return err
	}
	switch p.FirstByte() {
	case OK:
		if err = r.ReadOK(&p); err != nil {
			return err
		}
	case ERR:
		return p.ReadErr()
	default:
		return fmt.Errorf("sendLocalFile: expected OK or ERR, got %v", p.FirstByte())
	}

	return nil
}

func (cn *conn) logWarnings(warnings uint16) {
	if cn.debug && warnings > 0 {
		log.Printf("warnings: %d\n", warnings)
	}
}

func (st *stmt) Exec(args []driver.Value) (driver.Result, error) {
	if st.cn.debug {
		log.Println("exec:", st.qs, args)
	}
	return st.exec(args)
}

func (st *stmt) exec(args []driver.Value) (r *result, err error) {
	r, err = st.query(args)
	if err != nil {
		return nil, err
	}
	if err = r.Close(); err != nil {
		return nil, err
	}
	return r, nil
}

func (st *stmt) Query(args []driver.Value) (driver.Rows, error) {
	if st.cn.debug {
		log.Println("query:", st.qs, args)
	}
	return st.query(args)
}

func (st *stmt) sendLongData(paramId int, b *bytes.Buffer) error {
	for b.Len() > 0 {
		p := st.cn.newComPacket(COM_STMT_SEND_LONG_DATA)
		p.WriteUint32(st.stmtId)
		p.WriteUint16(uint16(paramId))
		p.Write(b.Next(MAX_DATA_CHUNK))
		if err := st.cn.sendPacket(p); err != nil {
			return err
		}
	}
	return nil
}

func (st *stmt) sendLongArgs(args []driver.Value) error {
	for i, a := range args {
		switch t := a.(type) {
		case []byte:
			if len(t) > MAX_DATA_CHUNK {
				return st.sendLongData(i, bytes.NewBuffer(t))
			}
		case string:
			if len(t) > MAX_DATA_CHUNK {
				return st.sendLongData(i, bytes.NewBufferString(t))
			}
		}
	}
	return nil
}

func (st *stmt) query(args []driver.Value) (r *result, err error) {
	if err = st.sendLongArgs(args); err != nil {
		return nil, err
	}

	p := st.cn.newComPacket(COM_STMT_EXECUTE)
	p.WriteUint32(st.stmtId)
	p.WriteByte(CURSOR_TYPE_NO_CURSOR)
	p.WriteUint32(1)
	if len(args) > 0 {
		nullMask := make([]bool, len(args))
		for i, a := range args {
			nullMask[i] = a == nil
		}
		p.WriteMask(nullMask)
		p.WriteByte(1)
		if err := p.WriteArgs(args); err != nil {
			return nil, err
		}
	}
	if err = st.cn.sendPacket(p); err != nil {
		return nil, err
	}

	r = &result{cn: st.cn, binary: true}

	if err = r.readResponse(); err != nil {
		return nil, err
	}

	return r, nil
}

func (st *stmt) NumInput() int {
	return len(st.params)
}

func (st *stmt) Close() error {
	if st.cn.debug {
		log.Println("close")
	}
	p := st.cn.newComPacket(COM_STMT_CLOSE)
	p.WriteUint32(st.stmtId)
	if err := st.cn.sendPacket(p); err != nil {
		return err
	}
	return nil
}

func (r *result) ReadOK(p *packet) error {
	r.rowsAffected, r.lastInsertId, r.warnings = p.ReadOK()
	r.closed = true
	return r.ReadWarnings()
}

func (r *result) readResponse() (err error) {
	p, err := r.cn.recvPacket()
	if err != nil {
		return err
	}

	switch {
	case p.FirstByte() == OK:
		if err = r.ReadOK(&p); err != nil {
			return err
		}
	case p.FirstByte() == ERR:
		r.closed = true
		err = p.ReadErr()
		if r.cn.debug {
			log.Printf("%v", err)
		}
		return err
	case r.binary == false && p.FirstByte() == LOCAL_INFILE:
		p.ReadUint8()
		fn := string(p.Bytes())
		if err = r.cn.sendLocalFile(r, fn); err != nil {
			return err
		}
	default:
		n, _ := p.ReadLCUint64()
		if r.columns, err = r.cn.readColumns(int(n)); err != nil {
			return err
		}
	}
	return nil
}

func (r *result) ReadWarnings() error {
	if r.warnings > 0 && (r.cn.strict || r.cn.debug) {
		w, err := r.cn.query("show warnings")
		if err != nil {
			return err
		}
		v := make([]driver.Value, 3)
		for {
			switch err := w.Next(v); err {
			case nil:
				if r.cn.debug {
					log.Printf("%s %s %s", v[0], v[1], v[2])
				}
				if r.cn.strict {
					if string(v[0].([]byte)) != "Note" {
						w.Close()
						return fmt.Errorf("%s %s %s", v[0], v[1], v[2])
					}
				}
			case io.EOF:
				return nil
			default:
				return err
			}
		}
	}
	return nil
}

func (r *result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

func (r *result) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

func (r *result) Columns() []string {
	c := make([]string, len(r.columns))
	for i, col := range r.columns {
		c[i] = col.name
	}
	return c
}

func (r *result) Close() error {
	for {
		err := r.Next(nil)
		switch err {
		case nil:
		case io.EOF:
			return nil
		default:
			return err
		}
	}
	panic("unreachable")
}

func (r *result) Next(dest []driver.Value) (err error) {
	if r.closed {
		return io.EOF
	}
	p, err := r.cn.recvPacket()
	if err != nil {
		return err
	}

	switch {
	case p.FirstByte() == ERR:
		return p.ReadErr()
	case p.FirstByte() == EOF && p.Len() <= 8: // can be LC integer
		r.warnings, r.status = p.ReadEOF()
		err := r.ReadWarnings()

		// Read anymore results, if multiple errors are encounterd return first
		// error.
		nerr := io.EOF
		if r.status&SERVER_MORE_RESULTS_EXISTS != 0 {
			nr := &result{cn: r.cn, binary: r.binary}
			nerr = nr.readResponse()
			if nr.closed == false {
				for {
					nerr = r.Next(nil)
					if nerr != nil {
						break
					}
				}
			} else if nerr == nil { // ok packet received with no strict errors
				nerr = io.EOF
			}

			if nr.rowsAffected != 0 || r.lastInsertId != 0 {
				r.rowsAffected = nr.rowsAffected
				r.lastInsertId = nr.lastInsertId
			}
			if nerr != io.EOF && err != nil {
				r.warnings = nr.warnings
			}
		}

		r.closed = true
		if err != nil {
			return err
		} else if nerr != io.EOF {
			return nerr
		}
		return io.EOF
	default:
		if r.binary {
			if h := p.ReadUint8(); h != 0 {
				return fmt.Errorf("next: expected 0, got %v", h)
			}
			nullMask := p.ReadMask(len(r.columns) + 2)
			nullMask = nullMask[2:]
			for i := range dest {
				dest[i], err = p.ReadValue(r.columns[i].coltype, r.columns[i].flags, nullMask[i])
				if err != nil {
					return err
				}
			}
		} else {
			for i := range dest {
				dest[i], err = p.ReadTextValue(r.columns[i].coltype, r.columns[i].flags)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
