// Copyright (c) Huawei Technologies Co., Ltd. 2025-2025. All rights reserved.

// Package models
package models

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/frontend/common/httpconstant"
)

// MultiGetSuccessResponse -
type MultiGetSuccessResponse struct {
	CommonRspHeader
	DataSystemPayloadInfo *DataSystemPayloadInfo
	RawData               [][]byte
}

// WriteResponse -
func (rsp *MultiGetSuccessResponse) WriteResponse(ctx *gin.Context) error {
	if (rsp.DataSystemPayloadInfo == nil && len(rsp.RawData) > 0) ||
		(rsp.DataSystemPayloadInfo != nil && len(rsp.RawData) == 0) {
		return fmt.Errorf("payload incomplete: missing PayloadInfo or RawData")
	}

	ctx.Writer.WriteHeader(statuscode.Code(int(rsp.InnerCode)))
	ctx.Writer.Header().Set(constant.HeaderContentType, httpconstant.StreamContentType)
	if rsp.DataSystemPayloadInfo != nil {
		ctx.Writer.Header().Set(constant.HeaderDataSystemPayloadInfo, rsp.DataSystemPayloadInfo.ToJSON())
	}

	ctx.Writer.Header().Set(constant.HeaderInnerCode, fmt.Sprint(rsp.InnerCode))
	ctx.Writer.Header().Set(constant.HeaderTraceID, fmt.Sprint(rsp.TraceID))
	for _, data := range rsp.RawData {
		_, err := ctx.Writer.Write(data)
		if err != nil {
			log.GetLogger().Errorf("failed to write rsp err %s", err.Error())
			return err
		}
	}
	return nil
}
