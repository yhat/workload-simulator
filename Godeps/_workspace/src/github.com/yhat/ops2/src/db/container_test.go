package db

import (
	"database/sql"
	"fmt"
	"sort"
	"testing"
)

func TestNewDeployment(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		user, err := NewUser(tx, "bigdatabob", "foo", "", true)
		if err != nil {
			t.Errorf("could not create user: %v", err)
			return
		}

		seen := map[int64]bool{}
		models := []string{"hellopy_1", "hellopy_2", "hellopy_3", "hellopy_4"}
		for _, model := range models {
			params := NewVersionParams{
				UserId:         user.Id,
				Model:          model,
				Lang:           LangPython2,
				BundleFilename: "bundle.json",
			}
			if _, err := NewModelVersion(tx, &params); err != nil {
				t.Errorf("could not create new model version: %v", err)
				return
			}
			nInstances := 3

			for i := 0; i < 5; i++ {
				deployid, instIds, err := NewDeployment(tx, "bigdatabob", params.Model, i, nInstances)
				if err != nil {
					t.Errorf("could not create new deployment: %v", err)
					return
				}
				if exp, got := nInstances, len(instIds); exp != got {
					t.Errorf("expected NewDeployment to return %d instance ids got %d", exp, got)
					return
				}
				if seen[deployid] {
					t.Errorf("got the same deployid more than once: %d", deployid)
				}
				seen[deployid] = true
			}
		}

	})
}

func TestDeploymentRequests(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		user, err := NewUser(tx, "bigdatabob", "foo", "", true)
		if err != nil {
			t.Errorf("could not create user: %v", err)
			return
		}

		models := []string{"hellopy_1", "hellopy_2", "hellopy_3", "hellopy_4"}
		for _, model := range models {
			params := NewVersionParams{
				UserId:         user.Id,
				Model:          model,
				Lang:           LangPython2,
				BundleFilename: "bundle.json",
			}
			if _, err := NewModelVersion(tx, &params); err != nil {
				t.Errorf("could not create new model version: %v", err)
				return
			}

			for i := 0; i < 5; i++ {
				_, _, err := NewDeployment(tx, "bigdatabob", params.Model, i, 3)
				if err != nil {
					t.Errorf("failed to request deployment: %v", err)
					continue
				}
			}
		}
		reqs, err := DeploymentRequests(tx)
		if err != nil {
			t.Errorf("failed to make deployment requests: %v", err)
			return
		}
		if got, exp := len(reqs), len(models); got != exp {
			t.Errorf("expected %d deployment request, got %d", exp, got)
		}
	})
}

type intSlice []int64

func (p intSlice) Len() int           { return len(p) }
func (p intSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p intSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func compareInt64(s1, s2 []int64) error {
	if len(s1) != len(s2) {
		return fmt.Errorf("lengths did not match %d vs %d", len(s1), len(s2))
	}
	sort.Sort(intSlice(s1))
	sort.Sort(intSlice(s2))
	for i, val1 := range s1 {
		if val1 != s2[i] {
			return fmt.Errorf("slices did not match")
		}
	}

	return nil
}
