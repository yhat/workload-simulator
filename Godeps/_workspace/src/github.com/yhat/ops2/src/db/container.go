package db

import (
	"database/sql"
	"fmt"
)

type DeploymentReq struct {
	Username     string
	Modelname    string
	Version      int
	LastDeployId int64
	// if the model is asleep the supervisor should not attempt to redeploy
	Asleep           bool
	ValidInstanceIds []int64
}

func DeploymentRequests(tx *sql.Tx) ([]DeploymentReq, error) {
	q := `SELECT
	m.modelname, u.username, m.active_version, m.deployment_id, m.status, c.id
	FROM User u
	INNER JOIN Model m
	ON u.user_id = m.user_id
	INNER JOIN Container c
	ON m.deployment_id = c.deploy_id;`

	reqs := []DeploymentReq{}
	rows, err := tx.Query(q)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	for rows.Next() {
		d := DeploymentReq{}
		var instId int64
		var status string
		err := rows.Scan(&d.Modelname, &d.Username, &d.Version, &d.LastDeployId, &status, &instId)
		if err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		// only way we can test if a model is asleep
		// TODO: add an "asleep" field
		if status == "asleep" {
			d.Asleep = true
		}

		found := false
		for i, req := range reqs {
			if req.LastDeployId == d.LastDeployId {
				req.ValidInstanceIds = append(req.ValidInstanceIds, instId)
				found = true
				reqs[i] = req
				break
			}
		}

		if !found {
			d.ValidInstanceIds = []int64{instId}
			reqs = append(reqs, d)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sql error: %v", err)
	}
	return reqs, nil
}

func NewDeployment(tx *sql.Tx, user, model string, version, nInstances int) (
	deployId int64, instIds []int64, err error) {

	err = tx.QueryRow(`CALL NewDeployment(?, ?, ?)`, user, model, version).Scan(&deployId)
	if err != nil {
		return 0, nil, fmt.Errorf("query error: %v", err)
	}
	instIds = make([]int64, nInstances)
	for i := 0; i < nInstances; i++ {
		result, err := tx.Exec(`INSERT INTO Container (deploy_id) VALUES (?);`, deployId)
		if err != nil {
			return 0, nil, fmt.Errorf("could not reserve id: %v", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return 0, nil, fmt.Errorf("could not get last inserted id: %v", err)
		}
		instIds[i] = id
	}
	return
}
