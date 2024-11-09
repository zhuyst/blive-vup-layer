package dao

import (
	"context"
	"errors"
	"gorm.io/gorm"
)

type User struct {
	OpenID                 string `json:"open_id" gorm:"column:open_id;primarykey"`
	FansMedalWearingStatus bool   `json:"fans_medal_wearing_status" gorm:"column:fans_medal_wearing_status"`
	FansMedalLevel         int    `json:"fans_medal_level" gorm:"column:fans_medal_level"`
	GuardLevel             int    `json:"guard_level" gorm:"column:guard_level"`
}

func (User) TableName() string {
	return "user"
}

func (d *Dao) GetUser(ctx context.Context, openId string) (*User, error) {
	d.userMapMutex.RLock()
	user, ok := d.userMap[openId]
	d.userMapMutex.RUnlock()

	if ok {
		return user, nil
	}

	user = &User{OpenID: openId}
	err := d.db.WithContext(ctx).
		Where("open_id = ?", openId).
		First(&user).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		user = nil
	}

	d.userMapMutex.Lock()
	d.userMap[openId] = user
	d.userMapMutex.Unlock()

	return user, nil
}

func (d *Dao) CreateOrUpdateUser(ctx context.Context, user *User) error {
	d.userMapMutex.RLock()
	userInMap, ok := d.userMap[user.OpenID]
	d.userMapMutex.RUnlock()

	if ok && userInMap != nil && user.GuardLevel == userInMap.GuardLevel &&
		user.FansMedalWearingStatus == userInMap.FansMedalWearingStatus &&
		user.FansMedalLevel == userInMap.FansMedalLevel {
		return nil
	}

	d.userMapMutex.Lock()
	d.userMap[user.OpenID] = user
	d.userMapMutex.Unlock()

	return d.db.WithContext(ctx).
		Where("open_id = ?", user.OpenID).
		Assign(user).
		FirstOrCreate(user).Error
}
