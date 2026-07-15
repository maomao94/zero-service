package tool

import (
	"context"
	"reflect"
	"zero-service/common/ctxdata"
)

// GetCurrentUserId resolves the current user id from context data or a user object.
func GetCurrentUserId(ctx context.Context, currentUser interface{}) string {
	if userId := ctxdata.GetUserId(ctx); userId != "" {
		return userId
	}
	if currentUser != nil {
		v := reflect.ValueOf(currentUser)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return ""
		}
		userIdField := v.FieldByName("UserId")
		if !userIdField.IsValid() {
			return ""
		}
		switch userIdField.Kind() {
		case reflect.String:
			return userIdField.String()
		default:
			return ""
		}
	}
	return ""
}

// GetCurrentUserName resolves the current user name from context data or a user object.
func GetCurrentUserName(ctx context.Context, currentUser interface{}) string {
	if userName := ctxdata.GetUserName(ctx); userName != "" {
		return userName
	}
	if currentUser != nil {
		v := reflect.ValueOf(currentUser)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return ""
		}
		userNameField := v.FieldByName("UserName")
		if !userNameField.IsValid() {
			return ""
		}
		switch userNameField.Kind() {
		case reflect.String:
			return userNameField.String()
		default:
			return ""
		}
	}
	return ""
}

// GetCurrentDeptCode resolves the first department code from context data or a user object.
func GetCurrentDeptCode(ctx context.Context, currentUser interface{}) string {
	if deptCode := ctxdata.GetDeptCode(ctx); deptCode != "" {
		return deptCode
	}
	if currentUser != nil {
		v := reflect.ValueOf(currentUser)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return ""
		}
		deptField := v.FieldByName("Dept")
		if !deptField.IsValid() {
			return ""
		}
		if deptField.Kind() != reflect.Slice && deptField.Kind() != reflect.Array {
			return ""
		}
		if deptField.Len() == 0 {
			return ""
		}
		firstDept := deptField.Index(0)
		if firstDept.Kind() == reflect.Ptr {
			firstDept = firstDept.Elem()
		}
		deptCodeField := firstDept.FieldByName("DeptCode")
		if !deptCodeField.IsValid() || deptCodeField.Kind() != reflect.String {
			return ""
		}
		return deptCodeField.String()
	}
	return ""
}
