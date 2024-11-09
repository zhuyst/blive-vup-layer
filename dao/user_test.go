package dao

import (
	"context"
	"testing"
)

func TestGetUser(t *testing.T) {
	d, err := NewDao(MemoryFilePath)
	if err != nil {
		t.Errorf("NewDao err: %v", err)
		return
	}

	const openId = "test"
	if err := d.CreateOrUpdateUser(context.Background(), &User{
		OpenID:                 openId,
		FansMedalWearingStatus: true,
		FansMedalLevel:         15,
		GuardLevel:             1,
	}); err != nil {
		t.Errorf("CreateOrUpdateUser err: %v", err)
		return
	}

	u, err := d.GetUser(context.Background(), openId)
	if err != nil {
		t.Errorf("GetUser err: %v", err)
		return
	}
	t.Logf("open_id: %s", u.OpenID)
}
