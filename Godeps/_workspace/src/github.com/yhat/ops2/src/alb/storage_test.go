package alb

import (
	"fmt"
	"log"
	"strings"

	"github.com/yhat/ops2/src/mps"
)

var _ ModelStorage = &mockStorage{}

func NewTestStorage() *mockStorage {
	return &mockStorage{}
}

type mockStorage struct {
	modelRep     int
	nextDeployId int64

	nextid int64
	logger *log.Logger

	newDeployment func(user, model string, version int) (deployId int64, instIds []int64, err error)
}

func (*mockStorage) GetLatest(user, model string) (version int, err error) {
	return 1, nil
}
func (s *mockStorage) SetBuildStatus(user, model, status string) error {
	if s.logger != nil {
		s.logger.Printf("%s' model %s set to build status %s", user, model, status)
	}
	return nil
}
func (m *mockStorage) Get(user, model string, ver int) (*mps.DeployInfo, string, error) {
	info := &mps.DeployInfo{
		Username:  user,
		Modelname: model,
		Version:   ver,
	}

	switch {
	case strings.HasPrefix(model, "hellor"):
		info.Lang = mps.R
		return info, "../mps/bundles/r-bundle.json", nil
	case strings.HasPrefix(model, "hellopy"):
		info.Lang = mps.Python2
		return info, "../mps/bundles/py-bundle.json", nil
	default:
		return nil, "", fmt.Errorf("no version found!")
	}
}

func (m *mockStorage) NewDeployment(user, model string, version int) (deployId int64, instIds []int64, err error) {
	if m.newDeployment != nil {
		return m.newDeployment(user, model, version)
	}
	n := m.modelRep
	if n == 0 {
		// default to 2 models
		n = 2
	}
	instIds = make([]int64, n)
	for i := range instIds {
		instIds[i] = m.nextid
		m.nextid++
	}
	deployId = m.nextDeployId
	m.nextDeployId++
	return

}
