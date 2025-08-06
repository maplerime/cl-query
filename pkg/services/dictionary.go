package services

import (
	"context"
	"fmt"
	"web/src/model"

	"github.com/maplerime/cl-query/pkg/dbs"

	. "github.com/maplerime/cl-query/pkg/common"
)

type DictionaryAdmin struct{}

func (a *DictionaryAdmin) Get(ctx context.Context, id int64) (*model.Dictionary, error) {
	logger.Debugf("Enter DictionaryAdmin.Get, id=%d", id)
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	dictionary := &model.Dictionary{Model: model.Model{ID: id}}
	if err := db.Where(where).First(dictionary, id).Error; err != nil {
		logger.Debugf("DictionaryAdmin.Get: failed to get dictionary, id=%d, err=%v", id, err)
		logger.Debugf("Exit DictionaryAdmin.Get with error")
		return nil, fmt.Errorf("failed to get dictionary: %w", err)
	}
	logger.Debugf("DictionaryAdmin.Get: success, uuid=%s, dictionary=%+v", dictionary.UUID, dictionary)
	return dictionary, nil
}

func (a *DictionaryAdmin) List(ctx context.Context, offset, limit int64, order string, query string) (total int64, dictionaries []*model.Dictionary, err error) {
	logger.Debugf("Enter DictionaryAdmin.List, offset=%d, limit=%d, order=%s, query=%s", offset, limit, order, query)
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	dictionaries = []*model.Dictionary{}
	if err = db.Model(&model.Dictionary{}).Where(where).Where(query).Count(&total).Error; err != nil {
		logger.Debugf("DictionaryAdmin.List: count error, err=%v", err)
		logger.Debugf("Exit DictionaryAdmin.List with error")
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Where(where).Where(query).Find(&dictionaries).Error; err != nil {
		logger.Errorf("DictionaryAdmin.List: find error, err=%v", err)
		return
	}
	logger.Debugf("DictionaryAdmin.List: success, total=%d, count=%d", total, len(dictionaries))
	return
}

func (a *DictionaryAdmin) GetDictionaryByUUID(ctx context.Context, uuID string) (dictionaries *model.Dictionary, err error) {
	logger.Debugf("Enter DictionaryAdmin.GetDictionaryByUUID, uuID=%s", uuID)
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	dictionaries = &model.Dictionary{}
	err = db.Where(where).Where("uuid = ?", uuID).Take(dictionaries).Error
	if err != nil {
		logger.Errorf("DictionaryAdmin.GetDictionaryByUUID: failed, uuID=%s, err=%v", uuID, err)
		return
	}
	logger.Debugf("DictionaryAdmin.GetDictionaryByUUID: success, dictionary=%+v", dictionaries)
	return
}

func (a *DictionaryAdmin) Find(ctx context.Context, category, value string) (dictionary *model.Dictionary, err error) {
	db := DB()
	dictionary = &model.Dictionary{}
	err = db.Where("category = ? AND value = ?", category, value).Take(dictionary).Error
	if err != nil {
		logger.Error("DictionaryAdmin.Find: failed to get dictionary", err)
		return
	}
	return
}
