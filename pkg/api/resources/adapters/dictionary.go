/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Dictionary resource adapter
 *
**/

package adapters

import (
	"context"
	"fmt"
	"strings"
	"web/src/model"

	"github.com/gin-gonic/gin"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/services"
)

type DictionaryResponse struct {
	*ResourceReference
	Category string `json:"category"`
	Value    string `json:"value"`
}

type DictionaryListResponse struct {
	Offset       int                   `json:"offset"`
	Total        int                   `json:"total"`
	Limit        int                   `json:"limit"`
	Dictionaries []*DictionaryResponse `json:"dictionaries"`
}

type DictionaryFilters struct {
	Category string `json:"category,omitempty" binding:"omitempty"`
	Name     string `json:"name,omitempty" binding:"omitempty"`
	Value    string `json:"value,omitempty" binding:"omitempty"`
	Keyword  string `json:"keyword,omitempty" binding:"omitempty"`
}

type DictionaryAdapter struct {
	BaseAdapter
	service *services.DictionaryAdmin
}

func NewDictionaryAdapter() *DictionaryAdapter {
	logger.Debug("Creating new Dictionary adapter")
	return &DictionaryAdapter{
		service: &services.DictionaryAdmin{},
	}
}

func (a *DictionaryAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing Dictionary filters: %+v", filtersMap)

	filters, err := ParseFilters[DictionaryFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	if filters.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = '%s'", filters.Category))
		logger.Debugf("Added category filter: %s", filters.Category)
	}

	if filters.Name != "" {
		conditions = append(conditions, fmt.Sprintf("name like '%%%s%%'", filters.Name))
		logger.Debugf("Added name filter: %s", filters.Name)
	}

	if filters.Value != "" {
		conditions = append(conditions, fmt.Sprintf("value like '%%%s%%'", filters.Value))
		logger.Debugf("Added value filter: %s", filters.Value)
	}

	if filters.Keyword != "" {
		conditions = append(conditions, fmt.Sprintf("(name like '%%%s%%' OR value like '%%%s%%')", filters.Keyword, filters.Keyword))
		logger.Debugf("Added keyword filter: %s", filters.Keyword)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

func (a *DictionaryAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Dictionary list query with request: %+v", req)

	ctx := c.Request.Context()

	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, err
	}

	total, dictionaries, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	resp := &DictionaryListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(dictionaries),
	}

	list := make([]*DictionaryResponse, resp.Limit)
	for i, d := range dictionaries {
		list[i], err = a.getDictionaryResponse(ctx, d)
		if err != nil {
			logger.Errorf("Failed to build dictionary response: %v", err)
			return nil, err
		}
	}
	resp.Dictionaries = list
	logger.Debugf("List dictionaries successfully, %+v", resp)
	return resp, nil
}

func (a *DictionaryAdapter) Get(c *gin.Context, id string) (interface{}, error) {
	logger.Debugf("Starting Dictionary get query with ID: %s", id)

	ctx := c.Request.Context()
	d, err := a.service.GetDictionaryByUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	return a.getDictionaryResponse(ctx, d)
}

func (a *DictionaryAdapter) getDictionaryResponse(ctx context.Context, d *model.Dictionary) (*DictionaryResponse, error) {
	owner := orgAdmin.GetOrgName(ctx, d.Owner)
	resp := &DictionaryResponse{
		ResourceReference: &ResourceReference{
			ID:        d.UUID,
			Name:      d.Name,
			Owner:     owner,
			CreatedAt: d.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: d.UpdatedAt.Format(TimeStringForMat),
		},
		Category: d.Category,
		Value:    d.Value,
	}
	return resp, nil
}
