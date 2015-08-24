package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/timob/go-mysql"
	"github.com/yhat/ops2/src/mps"
)

const dbDriver = "mysql"

// User type represents a row in the scienceops User table.
type User struct {
	Id             int64
	Name           string
	Password       string
	Email          string
	Admin          bool
	Active         bool
	Apikey         string
	ReadOnlyApikey string
}

func uuid() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func NewUser(tx *sql.Tx, name, hashedPass, email string, admin bool) (*User, error) {
	if name == "" || hashedPass == "" {
		return nil, fmt.Errorf("invalid parameters")
	}
	apikey, err := uuid()
	if err != nil {
		return nil, fmt.Errorf("could not generate apikey for user: %v", err)
	}
	readonly_apikey, err := uuid()
	if err != nil {
		return nil, fmt.Errorf("could not generate readonly_apikey for user: %v", err)
	}

	stmt := `INSERT INTO User
	(username, password, email, admin)
	VALUES (?, ?, ?, ?);`
	result, err := tx.Exec(stmt, name, hashedPass, email, admin)
	if err != nil {
		return nil, err
	}
	userId, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(`INSERT INTO ApiKey (user_id, apikey, read_only_apikey) VALUES (?, ?, ?);`,
		userId, apikey, readonly_apikey)
	if err != nil {
		return nil, err
	}
	return &User{Id: userId, Name: name, Password: hashedPass, Email: email,
		Admin: admin, Active: true, Apikey: apikey}, nil
}

func SetPass(tx *sql.Tx, name, hashedPass string) error {
	q := `UPDATE User SET password = ? WHERE username = ?;`
	if _, err := tx.Exec(q, hashedPass, name); err != nil {
		return fmt.Errorf("update query: %v", err)
	}
	return nil
}

//Model type represents a row in the scienceops Model table.
type Model struct {
	Id             int64
	Name           string
	NumVersions    int
	ActiveVersion  int
	LastDeployment int64
	LastUpdated    time.Time
	Status         string
	Owner          string
}

func GetModel(tx *sql.Tx, user, model string) (*Model, error) {
	q := `SELECT m.model_id, m.modelname, m.active_version, m.status, m.deployment_id, COUNT(*), MAX(v.created_at)
	FROM ModelVersion v
	INNER JOIN Model m
	ON m.model_id = v.model_id
	INNER JOIN User u
	ON m.user_id = u.user_id
	WHERE u.username = ? AND m.modelname = ?
	GROUP BY m.model_id;`
	m := Model{}
	err := tx.QueryRow(q, user, model).Scan(&m.Id, &m.Name, &m.ActiveVersion,
		&m.Status, &m.LastDeployment, &m.NumVersions, &m.LastUpdated)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// DeleteModel removes a model from the database. It returns a slice of
// the bundle file names associated with the model.
// It is the caller's responsibility to remove the bundles files.
func DeleteModel(tx *sql.Tx, user, model string) ([]string, error) {
	q := `SELECT v.bundle
	FROM ModelVersion v
	INNER JOIN Model m
	ON m.model_id = v.model_id
	INNER JOIN User u
	ON m.user_id = u.user_id
	WHERE u.username = ? AND m.modelname = ?
	GROUP BY m.model_id;`
	rows, err := tx.Query(q, user, model)
	if err != nil {
		return nil, fmt.Errorf("failed to query bundles: %v", err)
	}
	defer rows.Close()

	bundles := []string{}

	// attempt to clean up the bundles
	for rows.Next() {
		var bundlepath string
		if err := rows.Scan(&bundlepath); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}
		bundles = append(bundles, bundlepath)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to remove bundles: %v", err)
	}

	d := `DELETE m
	FROM Model m 
	INNER JOIN User u
	ON m.user_id = u.user_id
	WHERE u.username = ? AND m.modelname = ?;`
	result, err := tx.Exec(d, user, model)
	if err != nil {
		return nil, fmt.Errorf("failed to remove from db: %v", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("could not get number of rows affected: %v", err)
	}
	if n == 0 {
		return nil, sql.ErrNoRows
	}
	return bundles, nil
}

func GetModelById(tx *sql.Tx, id int64) (*Model, error) {
	q := `SELECT m.model_id, m.modelname, m.active_version, m.status, COUNT(*), MAX(v.created_at)
	FROM ModelVersion v
	INNER JOIN Model m
	ON m.model_id = v.model_id
	WHERE m.model_id = ?
	GROUP BY m.model_id;`
	m := Model{}
	err := tx.QueryRow(q, id).Scan(&m.Id, &m.Name, &m.ActiveVersion,
		&m.Status, &m.NumVersions, &m.LastUpdated)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type ModelVersion struct {
	ModelId        int64
	Name           string
	Version        int
	CreatedAt      time.Time
	Code           string
	Lang           string
	LangPackages   []mps.Package
	UbuntuPackages []mps.Package
	BundleFilename string
}

// DO NOT CHANGE THESE
const (
	LangPython2 = "python2"
	LangR       = "r"
	LangAptGet  = "apt-get"
)

type NewVersionParams struct {
	UserId         int64
	Model          string
	Lang           string
	LangPackages   []mps.Package
	UbuntuPackages []mps.Package
	SourceCode     string
	BundleFilename string
}

func NewModelVersion(tx *sql.Tx, p *NewVersionParams) (version int, err error) {
	// do input validation
	if (p.Lang != LangPython2) && (p.Lang != LangR) {
		return 0, errors.New("model language must be db.LangPython2 or db.LangR")
	}
	if p.UserId == 0 {
		return 0, errors.New("userid cannot be zero")
	}
	if p.Model == "" {
		return 0, errors.New("model name cannot be empty")
	}
	if p.BundleFilename == "" {
		return 0, errors.New("bundle filename cannot be empty")
	}

	// verisonId is the database's internal id for the model version
	var versionId int64

	// run stored query
	query := `CALL NewModelVersion(?, ?, ?, ?, ?);`
	row := tx.QueryRow(query, p.Model, p.UserId, p.Lang, p.SourceCode, p.BundleFilename)
	err = row.Scan(&versionId, &version)
	if err != nil {
		return 0, fmt.Errorf("failed to create new model version: %v", err)
	}

	// associate the packages with this version id
	s := `INSERT INTO Package (version_id, name, version, lang) VALUES (?, ?, ?, ?);`
	for _, pkg := range p.LangPackages {
		if _, err = tx.Exec(s, versionId, pkg.Name, pkg.Version, p.Lang); err != nil {
			return 0, fmt.Errorf("failed to insert package: %v", err)
		}
	}
	for _, pkg := range p.UbuntuPackages {
		if _, err = tx.Exec(s, versionId, pkg.Name, pkg.Version, LangAptGet); err != nil {
			return 0, fmt.Errorf("failed to insert package: %v", err)
		}
	}
	return version, nil
}

func GetModelVersion(tx *sql.Tx, user, model string, version int) (*ModelVersion, error) {
	row := tx.QueryRow(`CALL GetModelVersion(?, ?, ?)`, user, model, version)
	mv := ModelVersion{Name: model, Version: version}
	var id int64
	err := row.Scan(&id, &mv.ModelId, &mv.CreatedAt, &mv.Code, &mv.BundleFilename, &mv.Lang)
	if err != nil {
		if IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query model: %v", err)
	}

	q := `SELECT name, version, lang FROM Package WHERE version_id = ?`
	rows, err := tx.Query(q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		pkg := mps.Package{}
		var lang string
		if err := rows.Scan(&pkg.Name, &pkg.Version, &lang); err != nil {
			return nil, fmt.Errorf("could not scan row: %v", err)
		}
		switch lang {
		case mv.Lang:
			mv.LangPackages = append(mv.LangPackages, pkg)
		case LangAptGet:
			mv.UbuntuPackages = append(mv.UbuntuPackages, pkg)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("could not query model version packages: %v", err)
	}

	return &mv, nil
}

func GetModelVersions(tx *sql.Tx, user, model string) ([]*ModelVersion, error) {
	m, err := GetModel(tx, user, model)
	if err != nil {
		return nil, err
	}
	versions := make([]*ModelVersion, m.NumVersions)

	for i := 0; i < m.NumVersions; i++ {
		v := i + 1
		version, err := GetModelVersion(tx, user, model, v)
		if err != nil {
			return nil, err
		}
		versions[i] = version
	}
	return versions, nil
}

// GetLatestVersion gets the latest version of a given model
func GetLatestVersion(tx *sql.Tx, user, model string) (version int, err error) {
	q := `SELECT IFNULL(MAX(v.version), 0)
	FROM ModelVersion v
	INNER JOIN Model m
	ON m.model_id = v.model_id
	INNER JOIN User u
	ON m.user_id = u.user_id
	WHERE u.username = ? AND m.modelname = ?;`

	err = tx.QueryRow(q, user, model).Scan(&version)
	if err == nil && version == 0 {
		err = sql.ErrNoRows
	}
	return
}

func SetBuildStatus(tx *sql.Tx, user, model, status string) error {
	q := `UPDATE Model m
	INNER JOIN User u
	ON m.user_id = u.user_id
	SET m.status = ?
	WHERE u.username = ? AND m.modelname = ?;`

	result, err := tx.Exec(q, status, user, model)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil {
		switch n {
		case 0:
			err = sql.ErrNoRows
		case 1:
		default:
			err = fmt.Errorf("expected to update one row, affected: %v", n)
		}
	}
	return err
}

// DeleteUser removes a user from the database. It returns a slice of all
// bundles on the file system associated with that user.
func DeleteUser(tx *sql.Tx, user string) ([]string, error) {

	bundles := []string{}

	q := `SELECT v.bundle
	FROM ModelVersion v
	INNER JOIN Model m ON v.model_id = m.model_id
	INNER JOIN User u ON u.user_id = m.user_id
	WHERE u.username = ?;`
	rows, err := tx.Query(q, user)
	if err != nil {
		return nil, fmt.Errorf("failed to query for bundles: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bundle string
		if err := rows.Scan(&bundle); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		bundles = append(bundles, bundle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	e := `DELETE FROM User WHERE username = ?;`
	result, err := tx.Exec(e, user)
	if err != nil {
		return nil, fmt.Errorf("delete user query failed: %v", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, sql.ErrNoRows
	}
	return bundles, nil
}

type SharedUser struct {
	Id   int64
	Name string
}

// NewSqlDB opens a connection to a  mysql database using a connection string.
func NewSqlDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open(dbDriver, connStr)
	if err != nil {
		return nil, fmt.Errorf("could not clean database: %v", err)
	}
	return db, nil
}

// InitTables runs sql script to create tables in scienceops db. The sqlPath specifies a
// a file path to a .sql file that can be used to intialize a database. An existing
// scienceops schema will be dropped if dropSchema is true.
func InitTables(tx *sql.Tx, sqlDir string) error {
	fileInfos, err := ioutil.ReadDir(sqlDir)
	if err != nil {
		return err
	}

	fileFound := false
	for _, fi := range fileInfos {
		if fi.IsDir() {
			continue
		}
		name := fi.Name()
		if !strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, "test.sql") {
			continue
		}

		err = RunScript(tx, filepath.Join(sqlDir, name))
		if err != nil {
			return err
		}
		fileFound = true
	}
	if !fileFound {
		return fmt.Errorf("no sql files found in directory")
	}
	return nil
}

type Worker struct {
	Id   int64
	Host string
}

func NewWorker(tx *sql.Tx, host string) (*Worker, error) {
	q := `INSERT INTO MPS (hostname) VALUES (?);`
	result, err := tx.Exec(q, host)
	if err != nil {
		return nil, fmt.Errorf("failed to insert worker: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Worker{id, host}, nil
}

func Workers(tx *sql.Tx) ([]Worker, error) {
	workers := []Worker{}
	rows, err := tx.Query(`SELECT id, hostname FROM MPS;`)
	if err != nil {
		return nil, fmt.Errorf("failed to query worker nodes: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var w Worker
		if err := rows.Scan(&w.Id, &w.Host); err != nil {
			return nil, fmt.Errorf("error while scanning: %v", err)
		}
		workers = append(workers, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db query failed: %v", err)
	}
	return workers, nil
}

func (w *Worker) Remove(tx *sql.Tx) error {
	return RemoveWorker(tx, w.Id)
}

func RemoveWorker(tx *sql.Tx, workerId int64) error {
	result, err := tx.Exec(`DELETE FROM MPS WHERE id=?;`, workerId)
	if err != nil {
		return fmt.Errorf("failed to execute delete: %v", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to validate remove: %v", err)
	}
	if n == 0 {
		return fmt.Errorf("no worker to remove")
	}
	return nil
}

func DropTables(tx *sql.Tx) error {
	_, err := tx.Exec("DROP SCHEMA IF EXISTS scienceops")
	return err
}

func RunScript(tx *sql.Tx, filename string) error {
	sqlQuries, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	stms := strings.Split(string(sqlQuries), "//")
	for _, stm := range stms {
		stm = strings.TrimSpace(stm)
		if stm == "" || strings.HasPrefix(stm, "DELIMITER") {
			continue
		}
		if _, err := tx.Exec(stm); err != nil {
			return err
		}
	}
	return nil
}

func IsNotFound(err error) bool {
	return err == sql.ErrNoRows
}

// GetUser finds a user by ID or slug. Uses whichever field is not false.
func GetUser(tx *sql.Tx, username string) (*User, error) {
	u := &User{Name: username}
	err := tx.QueryRow(`CALL UserSelectName(?)`, username).Scan(
		&u.Id, &u.Name, &u.Password, &u.Email, &u.Admin, &u.Active, &u.Apikey, &u.ReadOnlyApikey)
	if err != nil {
		if IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("error finding user by username=%s: %v", username, err)
	}
	return u, nil
}

func MakeAdmin(tx *sql.Tx, user string) error {
	return setAdmin(tx, user, true)
}

func UnmakeAdmin(tx *sql.Tx, user string) error {
	q := `SELECT * FROM User WHERE admin = true AND username != ?;`
	rows, err := tx.Query(q, user)
	if err != nil {
		return fmt.Errorf("select query: %v", err)
	}
	foundAnotherAdmin := rows.Next()
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %v", err)
	}
	if !foundAnotherAdmin {
		return fmt.Errorf("could not remove admin privileges of last admin")
	}

	return setAdmin(tx, user, false)
}

func setAdmin(tx *sql.Tx, user string, admin bool) error {
	q := `UPDATE User SET admin = ? WHERE username = ?;`
	result, err := tx.Exec(q, admin, user)
	if err != nil {
		return fmt.Errorf("update query: %v", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %v", err)
	}
	if n == 0 {
		return fmt.Errorf("no rows affected")
	}
	return nil
}

// GetUserSelectAll runs a query to get all users from the scienceops db.
func AllUsers(tx *sql.Tx) ([]User, error) {
	rows, err := tx.Query(`CALL UserSelectAll()`)
	if err != nil {
		return nil, fmt.Errorf("error querying user table for all users: %v", err)
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		u := User{}
		err := rows.Scan(
			&u.Id, &u.Name, &u.Password, &u.Email, &u.Admin, &u.Active, &u.Apikey, &u.ReadOnlyApikey)
		if err != nil {
			return nil, fmt.Errorf("GetUserSelectAll: error scanning rows: %v", err)
		}
		users = append(users, u)

	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error encountered during iteration: %v", err)
	}
	return users, nil
}

func UserModels(tx *sql.Tx, username string) ([]Model, error) {

	rows, err := tx.Query(`CALL UserModels(?)`, username)
	if err != nil {
		return nil, fmt.Errorf("error querying model table: %v", err)
	}
	defer rows.Close()
	models := []Model{}

	for rows.Next() {
		m := Model{}
		err := rows.Scan(&m.Id, &m.Name, &m.NumVersions, &m.LastUpdated, &m.Status)

		if err != nil {
			return nil, fmt.Errorf("GetModelsByUser: error scanning rows: %v", err)
		}
		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to get models: %v", err)
	}

	return models, nil
}

type SharedModelInfo struct {
	Owner       string
	Name        string
	LastUpdated time.Time
}

type sharedByName []SharedModelInfo

func (s sharedByName) Len() int      { return len(s) }
func (s sharedByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s sharedByName) Less(i, j int) bool {
	if s[i].Owner == s[j].Owner {
		return s[i].Name < s[j].Name
	}
	return s[i].Owner < s[j].Owner
}

func SharedModels(tx *sql.Tx, username string) ([]SharedModelInfo, error) {
	q := `SELECT u.username, m.modelname, MAX(v.created_at)
	FROM User u
	INNER JOIN Model m
		ON u.user_id = m.user_id
	INNER JOIN ModelVersion v
		ON v.model_id = m.model_id
	INNER JOIN ModelSharedUser s
		ON s.model_id = m.model_id
	INNER JOIN User u2
		ON u2.user_id = s.shared_user_id
	WHERE u2.username = ?
	GROUP BY m.model_id;`

	rows, err := tx.Query(q, username)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	sharedModels := []SharedModelInfo{}
	for rows.Next() {
		info := SharedModelInfo{}
		if err := rows.Scan(&info.Owner, &info.Name, &info.LastUpdated); err != nil {
			return nil, fmt.Errorf("scan failed: %v", err)
		}
		sharedModels = append(sharedModels, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning rows: %v", err)
	}

	sort.Sort(sharedByName(sharedModels))

	return sharedModels, nil
}

func ModelSharedUsers(tx *sql.Tx, username, modelname string) ([]SharedUser, error) {
	// first get all users
	q1 := `SELECT 
	u.user_id, u.username 
	FROM User u
	INNER JOIN ModelSharedUser s
		ON u.user_id = shared_user_id
	INNER JOIN Model m
		ON m.model_id = s.model_id
	INNER JOIN User u2
		ON u2.user_id = m.user_id
	WHERE u2.username = ? AND m.modelname = ?;
	`
	users := []SharedUser{}

	rows, err := tx.Query(q1, username, modelname)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	for rows.Next() {
		u := SharedUser{}
		err := rows.Scan(&u.Id, &u.Name)
		if err != nil {
			return nil, fmt.Errorf("GetSharedUsersByModel: error scanning rows: %v", err)
		}
		users = append(users, u)
	}
	return users, nil
}

func sharedIds(tx *sql.Tx, owner, modelname, user string) (modelId, userId int64, err error) {

	q1 := `SELECT m.model_id
	FROM Model m
	INNER JOIN User u
	ON m.user_id = u.user_id
	WHERE u.username = ? AND m.modelname = ?`

	if err := tx.QueryRow(q1, owner, modelname).Scan(&modelId); err != nil {
		return 0, 0, fmt.Errorf("could not get model id: %v", err)
	}

	q2 := `SELECT user_id FROM User WHERE username = ?;`
	if err := tx.QueryRow(q2, user).Scan(&userId); err != nil {
		return 0, 0, fmt.Errorf("could not get user id: %v", err)
	}
	return modelId, userId, nil
}

func StartSharing(tx *sql.Tx, owner, modelname, user string) error {

	modelId, userId, err := sharedIds(tx, owner, modelname, user)
	if err != nil {
		return err
	}

	e := `INSERT INTO ModelSharedUser
	(model_id, shared_user_id)
	VALUES (?, ?);`
	result, err := tx.Exec(e, modelId, userId)
	if err != nil {
		return fmt.Errorf("could not share model: %v", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get number of rows affected: %v", err)
	}
	if n != 1 {
		return fmt.Errorf("expected one row affected, got %d", n)
	}
	return nil
}

func StopSharing(tx *sql.Tx, owner, modelname, user string) error {

	modelId, userId, err := sharedIds(tx, owner, modelname, user)
	if err != nil {
		return err
	}

	e := `DELETE FROM ModelSharedUser
	WHERE model_id = ? AND shared_user_id = ?;`
	result, err := tx.Exec(e, modelId, userId)
	if err != nil {
		return fmt.Errorf("could not share model: %v", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get number of rows affected: %v", err)
	}
	if n != 1 {
		return fmt.Errorf("expected one row affected, got %d", n)
	}
	return nil
}

func UpdateApiKey(tx *sql.Tx, user_id int64, apikey string, is_readonly bool) error {
	var q string
	if is_readonly {
		q = `Call UpdateReadOnlyApikey(?, ?)`
	} else {
		q = `Call UpdateApikey(?, ?)`
	}
	if _, err := tx.Exec(q, user_id, apikey); err != nil {
		return fmt.Errorf("Error updating the ApiKey table: %v", err)
	}

	return nil
}

func AddModelSharedUser(tx *sql.Tx, model_id int, user_id int) error {
	_, err := tx.Exec(`Call AddModelSharedUser(?, ?)`, model_id, user_id)
	if err != nil {
		return fmt.Errorf("Error the ModelSharedUser table: %v", err)
	}
	return nil
}

func RemoveModelSharedUser(tx *sql.Tx, model_id int, user_id int) error {
	_, err := tx.Exec(`Call RemoveModelSharedUser(?, ?)`, model_id, user_id)
	if err != nil {
		return fmt.Errorf("Error the ModelSharedUser table: %v", err)
	}

	return nil
}

func SetModelStatus(tx *sql.Tx, model_id int64, status string) error {
	_, err := tx.Exec(`Call SetModelStatus(?, ?)`, model_id, status)
	if err != nil {
		return fmt.Errorf("Error setting status for model: %v", err)
	}
	return nil
}

type Apikeys struct {
	Apikey   string
	ReadOnly string
}

type SharedModel struct {
	User  string
	Owner string
	Model string
}

type PredictionAuth struct {
	Users  map[string]Apikeys
	Shared []SharedModel
}

// GetAuth is used by the supervisor to determine which users have access
// to which models.
func GetAuth(tx *sql.Tx) (*PredictionAuth, error) {
	q1 := `SELECT u.username, k.apikey, k.read_only_apikey
	FROM User u
	INNER JOIN ApiKey k
	ON u.user_id = k.user_id;`

	rows, err := tx.Query(q1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make(map[string]Apikeys)

	for rows.Next() {
		apikeys := Apikeys{}
		var username string
		err := rows.Scan(&username, &apikeys.Apikey, &apikeys.ReadOnly)
		if err != nil {
			return nil, err
		}
		users[username] = apikeys
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	q2 := `SELECT shareduser.username, owner.username, m.modelname
	FROM ModelSharedUser s
	INNER JOIN User shareduser
	ON shareduser.user_id = s.shared_user_id
	INNER JOIN Model m
	ON m.model_id = s.model_id
	INNER JOIN User owner
	ON owner.user_id = m.user_id;`

	rows, err = tx.Query(q2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sharedModels := []SharedModel{}

	for rows.Next() {
		s := SharedModel{}
		err := rows.Scan(&s.User, &s.Owner, &s.Model)
		if err != nil {
			return nil, err
		}
		sharedModels = append(sharedModels, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &PredictionAuth{
		Users:  users,
		Shared: sharedModels,
	}, nil
}

// error will be sql.ErrNoRows if there is no value in the db
func GetBaseImage(tx *sql.Tx, lang string) (string, error) {
	query := `SELECT image_name
	FROM DockerBaseImages d
	WHERE d.lang = ?;`

	result := tx.QueryRow(query, lang)

	var baseImage string
	if err := result.Scan(&baseImage); err != nil {
		if err == sql.ErrNoRows {
			return "", err
		}
		return "", fmt.Errorf("failed to scan baseImage row: %v", err)
	}

	return baseImage, nil
}

func ModelExample(tx *sql.Tx, user, model string) (input string, err error) {
	q := `SELECT m.example_input
	FROM Model m
	INNER JOIN User u ON u.user_id = m.user_id
	WHERE m.modelname = ? AND u.username = ?;`
	err = tx.QueryRow(q, model, user).Scan(&input)
	return
}

func SetModelExample(tx *sql.Tx, user, model, input string) error {
	q := `UPDATE Model m
	INNER JOIN User u ON m.user_id = u.user_id 
	SET m.example_input = ?
	WHERE m.modelname = ? AND u.username = ?;`
	result, err := tx.Exec(q, input, model, user)
	if err != nil {
		return fmt.Errorf("query failed: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get number of affected rows: %v", err)
	}
	if n != 1 {
		return fmt.Errorf("expected 1 affected row got %d", n)
	}
	return nil
}
