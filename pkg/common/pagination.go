/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose: common parse pagination
 *
**/

package common

import (
	"strconv"
)

func ParsePagination(offsetStr, limitStr string) (offset, limit int) {
	offset, _ = strconv.Atoi(offsetStr)
	limit, _ = strconv.Atoi(limitStr)
	if offset <= 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	return
}
