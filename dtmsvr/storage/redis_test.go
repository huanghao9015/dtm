package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yedf/dtm/common"
	"github.com/yedf/dtm/dtmcli/dtmimp"
)

func TestMain(m *testing.M) {
	common.MustLoadConfig()
	GetStore().PopulateData(false)
	m.Run()
}

func initTransGlobal(gid string) (*TransGlobalStore, Store) {
	next := time.Now().Add(10 * time.Second)
	g := &TransGlobalStore{Gid: gid, Status: "prepared", NextCronTime: &next}
	bs := []TransBranchStore{
		{BranchID: "01"},
	}
	s := GetStore()
	err := s.SaveNewTrans(g, bs)
	dtmimp.E2P(err)
	return g, s
}

func TestRedisSave(t *testing.T) {
	bs := []TransBranchStore{
		{BranchID: "01"},
		{BranchID: "02"},
	}
	gid := dtmimp.GetFuncName()
	g, s := initTransGlobal(gid)
	g2 := &TransGlobalStore{}
	err := s.GetTransGlobal(gid, g2)
	assert.Nil(t, err)
	assert.Equal(t, gid, g2.Gid)

	bs2 := s.GetBranches(gid)
	assert.Equal(t, len(bs2), int(1))
	assert.Equal(t, "01", bs2[0].BranchID)

	err = s.LockGlobalSaveBranches(gid, g.Status, []TransBranchStore{bs[1]}, -1)
	assert.Nil(t, err)
	bs3 := s.GetBranches(gid)
	assert.Equal(t, 2, len(bs3))
	assert.Equal(t, "02", bs3[1].BranchID)
	assert.Equal(t, "01", bs3[0].BranchID)

	err = s.LockGlobalSaveBranches(g.Gid, "submitted", []TransBranchStore{bs[1]}, 1)
	assert.Equal(t, ErrNotFound, err)

	err = s.ChangeGlobalStatus(g, "succeed", []string{}, true)
	assert.Nil(t, err)
}

func TestRedisChangeStatus(t *testing.T) {
	gid := dtmimp.GetFuncName()
	g, s := initTransGlobal(gid)
	g.Status = "no"
	err := s.ChangeGlobalStatus(g, "submitted", []string{}, false)
	assert.Equal(t, ErrNotFound, err)
	g.Status = "prepared"
	err = s.ChangeGlobalStatus(g, "submitted", []string{}, false)
	assert.Nil(t, err)
	err = s.ChangeGlobalStatus(g, "succeed", []string{}, true)
}

func TestRedisLockTrans(t *testing.T) {
	gid := dtmimp.GetFuncName()
	g, s := initTransGlobal(gid)

	g2 := &TransGlobalStore{}
	err := s.LockOneGlobalTrans(g2, 2*time.Duration(config.RetryInterval)*time.Second)
	assert.Nil(t, err)
	assert.Equal(t, gid, g2.Gid)
	err = s.LockOneGlobalTrans(g2, 2*time.Duration(config.RetryInterval)*time.Second)
	assert.Nil(t, err)
	assert.Equal(t, gid, g2.Gid)
	err = s.ChangeGlobalStatus(g, "succeed", []string{}, true)
	assert.Nil(t, err)
	err = s.LockOneGlobalTrans(g2, 2*time.Duration(config.RetryInterval)*time.Second)
	assert.Equal(t, ErrNotFound, err)
}
