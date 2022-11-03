// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

var (
	// Precompute the reflect.Type of error and http.Request
	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfRequest = reflect.TypeOf((*http.Request)(nil)).Elem()
)

// ----------------------------------------------------------------------------
// service
// ----------------------------------------------------------------------------

type service struct {
	name     string                    // name of service
	rcvr     reflect.Value             // receiver of methods for the service
	rcvrType reflect.Type              // type of the receiver
	methods  map[string]*serviceMethod // registered methods
	passReq  bool
}

type serviceMethod struct {
	method    reflect.Method // receiver method
	argsType  reflect.Type   // type of the request argument
	replyType reflect.Type   // type of the response argument
}

// ----------------------------------------------------------------------------
// serviceMap
// ----------------------------------------------------------------------------

// serviceMap is a registry for services.
type serviceMap struct {
	mutex    sync.Mutex
	services map[string]*service
	defaultService *service
}

// register adds a new service using reflection to extract its methods.
func (m *serviceMap) register(rcvr interface{}, name string, passReq,isDefault bool) error {
	// Setup service.
	s := &service{
		name:     name,
		rcvr:     reflect.ValueOf(rcvr),
		rcvrType: reflect.TypeOf(rcvr),
		methods:  make(map[string]*serviceMethod),
		passReq:  passReq,
	}
	if name == "" {
		s.name = reflect.Indirect(s.rcvr).Type().Name()
		if !isExported(s.name) {
			return fmt.Errorf("rpc: type %q is not exported", s.name)
		}
	}
	if s.name == "" {
		return fmt.Errorf("rpc: no service name for type %q",
			s.rcvrType.String())
	}
	// Setup methods.
	for i := 0; i < s.rcvrType.NumMethod(); i++ {

		method := s.rcvrType.Method(i)
		mtype := method.Type

		log.Printf("got method %s",method.Name)

		// offset the parameter indexes by one if the
		// service methods accept an HTTP request pointer
		var paramOffset int
		if passReq {
			paramOffset = 1
		} else {
			paramOffset = 0
		}

		// Method must be exported.
		if method.PkgPath != "" {

			log.Printf("got method %s is not exported skipping it",method.Name)
			continue
		}
		// Method needs four ins: receiver, *http.Request, *args, *reply.
		if mtype.NumIn() != 3+paramOffset {

			log.Printf("got method %s does not Method needs four ins: receiver, *http.Request, *args, *reply. skipping it",method.Name)
			continue
		}

		// If the service methods accept an HTTP request pointer
		if passReq {

			// First argument must be a pointer and must be http.Request.
			reqType := mtype.In(1)
			if reqType.Kind() != reflect.Ptr || reqType.Elem() != typeOfRequest {

				log.Printf("got method %s First argument is not a pointer and must be http.Request. skipping it",method.Name)
				continue
			}
		}
		// Next argument must be a pointer and must be exported.
		args := mtype.In(1 + paramOffset)
		if args.Kind() != reflect.Ptr || !isExportedOrBuiltin(args) {

			log.Printf("got method %s 1 Next argument must be a pointer and must be exported.. skipping it",method.Name)
			continue
		}

		// Next argument must be a pointer and must be exported.
		reply := mtype.In(2 + paramOffset)
		if reply.Kind() != reflect.Ptr || !isExportedOrBuiltin(reply) {

			log.Printf("got method %s 2 Next argument must be a pointer and must be exported.. skipping it",method.Name)
			continue
		}
		// Method needs one out: error.
		if mtype.NumOut() != 1 {

			log.Printf("got method %s Method needs one out: error. skipping it",method.Name)
			continue
		}

		if returnType := mtype.Out(0); returnType != typeOfError {

			log.Printf("got method %s return type is not error. skipping it",method.Name)
			continue
		}
		s.methods[method.Name] = &serviceMethod{
			method:    method,
			argsType:  args.Elem(),
			replyType: reply.Elem(),
		}
	}

	if len(s.methods) == 0 {

		return fmt.Errorf("rpc: %q has no exported methods of suitable type",
			s.name)
	}




	// Add to the map.
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if isDefault {

		m.defaultService = s
		return nil

	} else {

		if m.services == nil {

			m.services = make(map[string]*service)

		} else if _, ok := m.services[s.name]; ok {

			return fmt.Errorf("rpc: service already defined: %q", s.name)
		}
	}

	m.services[s.name] = s
	return nil
}

// get returns a registered service given a method name.
//
// The method name uses a dotted notation as in "Service.Method".
func (m *serviceMap) get(method string) (*service, *serviceMethod, error) {
	parts := strings.Split(method, ".")

	if len(parts) != 2 && len(parts) != 1 {
		err := fmt.Errorf("rpc: service/method request ill-formed: %q", method)
		return nil, nil, err
	}

	log.Printf("wants to look for method %s",method)

	m.mutex.Lock()

	var service *service

	if len(parts) == 1 {

		service = m.defaultService

	} else {

		service = m.services[parts[0]]

	}

	log.Printf("wants to look for method %s.%s",service.name,method)

	m.mutex.Unlock()

	if service == nil {

		err := fmt.Errorf("rpc: can't find service %q", method)
		return nil, nil, err
	}

	var serviceMethod *serviceMethod

	if len(parts) == 1 {

		serviceMethod = service.methods[parts[0]]

	} else {

		serviceMethod = service.methods[parts[1]]

	}

	if serviceMethod == nil {

		err := fmt.Errorf("rpc: can't find method %q", method)
		return nil, nil, err
	}
	return service, serviceMethod, nil
}

// isExported returns true of a string is an exported (upper case) name.
func isExported(name string) bool {
	inString, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(inString)
}

// isExportedOrBuiltin returns true if a type is exported or a builtin.
func isExportedOrBuiltin(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}
