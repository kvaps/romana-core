// Copyright (c) 2015 Pani Networks
// All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package common

import (
	"fmt"
	"net/http"
)

// Error is a structure that represents an error.
type HttpError struct {
	StatusCode int    `json:"status_code"`
	StatusText string `json:"status_text"`
	Message    string `json:"message"`
}

func NewError500(err error) HttpError {
	return NewError(http.StatusInternalServerError, err.Error())
}

func NewError400(err error, request string) HttpError {
	msg := fmt.Sprintf("Error parsing request \"%s\": %s", request, err.Error())
	return NewError(http.StatusBadRequest, msg)
}

// NewError helps to construct new Error structure.
func NewError(code int, msg string) HttpError {
	return HttpError{
		StatusCode: code,
		StatusText: http.StatusText(code),
		Message:    msg,
	}
}

// Error is a method to satisfy error interface and returns a string representation of the error.
func (e HttpError) Error() string {
	return fmt.Sprintf("%d %s %s", e.StatusCode, e.StatusText, e.Message)
}
