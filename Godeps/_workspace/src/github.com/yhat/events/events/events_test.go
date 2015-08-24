package events

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/yhat/events/events/oldclient"
)

func runDBTest(t *testing.T, test func(connStr string)) {
	cmd := exec.Command("docker", "run",
		"-p", "5432:5432",
		"-e", "POSTGRES_PASSWORD=pass",
		"-d", "postgres:9.4.2")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("creating container %v: %s", err, out)
		return
	}
	defer func() {
		cid := strings.TrimSpace(string(out))
		out, err = exec.Command("docker", "rm", "-f", cid).CombinedOutput()
		if err != nil {
			t.Errorf("removing containter: %v", err)
		}
	}()

	connStr := "postgres://postgres:pass@0.0.0.0:5432?sslmode=disable"
	for i := 0; i < 10; i++ {
		time.Sleep(2 * time.Second)
		var db *sql.DB
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			continue
		}
		if err = db.Ping(); err != nil {
			continue
		} else {
			break
		}
	}
	if err != nil {
		t.Errorf("could not connect to db: %v", err)
		return
	}
	test(connStr)
}

func runTest(t *testing.T, test func(ev *EventLogger)) {
	runDBTest(t, func(connStr string) {
		ev, err := NewEventLogger(connStr)
		if err != nil {
			t.Error(err)
			return
		}
		test(ev)
	})
}

func TestNewLogger(t *testing.T) {
	runDBTest(t, func(connStr string) {
		_, err := NewEventLogger(connStr)
		if err != nil {
			t.Error(err)
			return
		}
		// tests create the database twice
		_, err = NewEventLogger(connStr)
		if err != nil {
			t.Error(err)
		}
	})
}

func TestSaveDeployment(t *testing.T) {
	runTest(t, func(ev *EventLogger) {
		start := time.Now()
		end := start.Add(3 * time.Second)
		d := &Deployment{
			StartTime: start.Unix(),
			EndTime:   end.Unix(),
			Username:  "bigdatabob",
			ModelName: "hellor",
			ModelLang: "python",
			ModelDeps: "foo==2131",
			ModelVer:  4,
			ModelSize: 123,
			ClientIP:  "10.0.0.1",
			Service:   "sandbox",
		}
		if err := ev.Save(d); err != nil {
			t.Error(err)
		}
	})
}

func TestOldClient(t *testing.T) {
	runTest(t, func(ev *EventLogger) {
		info := &oldclient.ModelInfo{
			ModelName: "hellor",
			ModelLang: "r",
			ModelVer:  8,
			ModelSize: 12321,
		}
		s := httptest.NewServer(http.HandlerFunc(ev.HandleV1))
		defer s.Close()

		start := time.Now()
		end := start.Add(3 * time.Second)

		if err := info.SendDeploy(s.URL, "bigdatabob", start, end); err != nil {
			t.Error(err)
		}
	})
}
