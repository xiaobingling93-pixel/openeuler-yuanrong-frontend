/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2025. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package middleware

import "frontend/pkg/frontend/types"

// HandlerChain -
type HandlerChain interface {
	Use(mws ...Middleware)
	Handle(ctx *types.InvokeProcessContext) error
}

// baseInvoker -
type baseHandler struct {
	handler Handler
}

// NewBaseHandler -
func NewBaseHandler(handler Handler) HandlerChain {
	return &baseHandler{
		handler: handler,
	}
}

// Use middlewares
func (bi *baseHandler) Use(mws ...Middleware) {
	for i := len(mws) - 1; i >= 0; i-- {
		bi.handler = mws[i](bi.handler)
	}
}

// Handle -
func (bi *baseHandler) Handle(ctx *types.InvokeProcessContext) error {
	return bi.handler(ctx)
}
