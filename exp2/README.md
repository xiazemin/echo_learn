https://github.com/labstack/echo/issues/1337


https://echo.labstack.com/guide/binding/

https://github.com/labstack/echo/issues/1337


v3.3.10

e.GET("/users/:name", func(c echo.Context) error {
		u := new(User)
		u.Name = c.Param("name")
		if err := c.Bind(u); err != nil {
			return c.JSON(http.StatusBadRequest, nil)
		}
		return c.JSON(http.StatusOK, u)
	})


func (e *Echo) Add(method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	name := handlerName(handler)
	e.router.Add(method, path, func(c Context) error {
		h := handler
		// Chain middleware
		for i := len(middleware) - 1; i >= 0; i-- {
			h = middleware[i](h)
		}
		return h(c)
	})
	r := &Route{
		Method: method,
		Path:   path,
		Name:   name,
	}
	e.router.routes[method+path] = r
	return r
}

func (r *Router) Add(method, path string, h HandlerFunc) {
	// Validate path
	if path == "" {
		panic("echo: path cannot be empty")
	}
	if path[0] != '/' {
		path = "/" + path
	}
	pnames := []string{} // Param names
	ppath := path        // Pristine path

	for i, l := 0, len(path); i < l; i++ {
		if path[i] == ':' {
			j := i + 1

			r.insert(method, path[:i], nil, skind, "", nil)
			for ; i < l && path[i] != '/'; i++ {
			}

			pnames = append(pnames, path[j:i])
			path = path[:j] + path[i:]
			i, l = j, len(path)

			if i == l {
				r.insert(method, path[:i], h, pkind, ppath, pnames)
				return
			}
			r.insert(method, path[:i], nil, pkind, "", nil)
		} else if path[i] == '*' {
			r.insert(method, path[:i], nil, skind, "", nil)
			pnames = append(pnames, "*")
			r.insert(method, path[:i+1], h, akind, ppath, pnames)
			return
		}
	}

	r.insert(method, path, h, skind, ppath, pnames)
}



func (r *Router) insert(method, path string, h HandlerFunc, t kind, ppath string, pnames []string) {
	// Adjust max param
	l := len(pnames)
	if *r.echo.maxParam < l {
		*r.echo.maxParam = l
	}

	cn := r.tree // Current node as root
	if cn == nil {
		panic("echo: invalid method")
	}
	search := path

	for {
		sl := len(search)
		pl := len(cn.prefix)
		l := 0

		// LCP
		max := pl
		if sl < max {
			max = sl
		}
		for ; l < max && search[l] == cn.prefix[l]; l++ {
		}

		if l == 0 {
			// At root node
			cn.label = search[0]
			cn.prefix = search
			if h != nil {
				cn.kind = t
				cn.addHandler(method, h)
				cn.ppath = ppath
				cn.pnames = pnames
			}
		} else if l < pl {
			// Split node
			n := newNode(cn.kind, cn.prefix[l:], cn, cn.children, cn.methodHandler, cn.ppath, cn.pnames)

			// Reset parent node
			cn.kind = skind
			cn.label = cn.prefix[0]
			cn.prefix = cn.prefix[:l]
			cn.children = nil
			cn.methodHandler = new(methodHandler)
			cn.ppath = ""
			cn.pnames = nil

			cn.addChild(n)

			if l == sl {
				// At parent node
				cn.kind = t
				cn.addHandler(method, h)
				cn.ppath = ppath
				cn.pnames = pnames
			} else {
				// Create child node
				n = newNode(t, search[l:], cn, nil, new(methodHandler), ppath, pnames)
				n.addHandler(method, h)
				cn.addChild(n)
			}
		} else if l < sl {
			search = search[l:]
			c := cn.findChildWithLabel(search[0])
			if c != nil {
				// Go deeper
				cn = c
				continue
			}
			// Create child node
			n := newNode(t, search, cn, nil, new(methodHandler), ppath, pnames)
			n.addHandler(method, h)
			cn.addChild(n)
		} else {
			// Node already exists
			if h != nil {
				cn.addHandler(method, h)
				cn.ppath = ppath
				if len(cn.pnames) == 0 { // Issue #729
					cn.pnames = pnames
				}
			}
		}
		return
	}
}


func (e *Echo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Acquire context
	c := e.pool.Get().(*context)
	c.Reset(r, w)

	h := NotFoundHandler

	if e.premiddleware == nil {
		e.router.Find(r.Method, getPath(r), c)
		h = c.Handler()
		for i := len(e.middleware) - 1; i >= 0; i-- {
			h = e.middleware[i](h)
		}
	} else {
		h = func(c Context) error {
			e.router.Find(r.Method, getPath(r), c)
			h := c.Handler()
			for i := len(e.middleware) - 1; i >= 0; i-- {
				h = e.middleware[i](h)
			}
			return h(c)
		}
		for i := len(e.premiddleware) - 1; i >= 0; i-- {
			h = e.premiddleware[i](h)
		}
	}

	// Execute chain
	if err := h(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}

	// Release context
	e.pool.Put(c)
}


func getPath(r *http.Request) string {
	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}
	return path
}


func (r *Router) Find(method, path string, c Context) {
	ctx := c.(*context)
	ctx.path = path
	cn := r.tree // Current node as root

	var (
		search  = path
		child   *node         // Child node
		n       int           // Param counter
		nk      kind          // Next kind
		nn      *node         // Next node
		ns      string        // Next search
		pvalues = ctx.pvalues // Use the internal slice so the interface can keep the illusion of a dynamic slice
	)

	// Search order static > param > any
	for {
		if search == "" {
			break
		}

		pl := 0 // Prefix length
		l := 0  // LCP length

		if cn.label != ':' {
			sl := len(search)
			pl = len(cn.prefix)

			// LCP
			max := pl
			if sl < max {
				max = sl
			}
			for ; l < max && search[l] == cn.prefix[l]; l++ {
			}
		}

		if l == pl {
			// Continue search
			search = search[l:]
		} else {
			cn = nn
			search = ns
			if nk == pkind {
				goto Param
			} else if nk == akind {
				goto Any
			}
			// Not found
			return
		}

		if search == "" {
			break
		}

		// Static node
		if child = cn.findChild(search[0], skind); child != nil {
			// Save next
			if cn.prefix[len(cn.prefix)-1] == '/' { // Issue #623
				nk = pkind
				nn = cn
				ns = search
			}
			cn = child
			continue
		}

		// Param node
	Param:
		if child = cn.findChildByKind(pkind); child != nil {
			// Issue #378
			if len(pvalues) == n {
				continue
			}

			// Save next
			if cn.prefix[len(cn.prefix)-1] == '/' { // Issue #623
				nk = akind
				nn = cn
				ns = search
			}

			cn = child
			i, l := 0, len(search)
			for ; i < l && search[i] != '/'; i++ {
			}
			pvalues[n] = search[:i]
			n++
			search = search[i:]
			continue
		}

		// Any node
	Any:
		if cn = cn.findChildByKind(akind); cn == nil {
			if nn != nil {
				cn = nn
				nn = cn.parent // Next (Issue #954)
				search = ns
				if nk == pkind {
					goto Param
				} else if nk == akind {
					goto Any
				}
			}
			// Not found
			return
		}
		pvalues[len(cn.pnames)-1] = search
		break
	}

	ctx.handler = cn.findHandler(method)
	ctx.path = cn.ppath
	ctx.pnames = cn.pnames

	// NOTE: Slow zone...
	if ctx.handler == nil {
		ctx.handler = cn.checkMethodNotAllowed()

		// Dig further for any, might have an empty value for *, e.g.
		// serving a directory. Issue #207.
		if cn = cn.findChildByKind(akind); cn == nil {
			return
		}
		if h := cn.findHandler(method); h != nil {
			ctx.handler = h
		} else {
			ctx.handler = cn.checkMethodNotAllowed()
		}
		ctx.path = cn.ppath
		ctx.pnames = cn.pnames
		pvalues[len(cn.pnames)-1] = ""
	}

	return
}

func (n *node) findChild(l byte, t kind) *node {
	for _, c := range n.children {
		if c.label == l && c.kind == t {
			return c
		}
	}
	return nil
}


func (e *Echo) Add(method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	e.router.routes[method+path] = r
	printTree(e.router.tree)
	return r
}

func printTree(tree *node) {
	v1, err1 := json.Marshal(struct {
		Kind          kind
		Label         byte
		Prefix        string
		Parent        *node
		Children      children
		ChildrenNum   int
		Ppath         string
		Pnames        []string
		MethodHandler *methodHandler
	}{
		Kind:          tree.kind,
		Label:         tree.label,
		Prefix:        tree.prefix,
		Parent:        tree.parent,
		Children:      tree.children,
		ChildrenNum:   len(tree.children),
		Ppath:         tree.ppath,
		Pnames:        tree.pnames,
		MethodHandler: tree.methodHandler,
	})
	fmt.Println(string(v1), err1)
	for i, v := range tree.children {
		fmt.Println(i)
		printTree(v)
	}
}

{"Kind":0,"Label":47,"Prefix":"/users/","Parent":null,"Children":[{}],"ChildrenNum":1,"Ppath":"","Pnames":null,"MethodHandler":{}} <nil>
0
{"Kind":1,"Label":58,"Prefix":":","Parent":{},"Children":null,"ChildrenNum":0,"Ppath":"/users/:name","Pnames":["name"],"MethodHandler":{}} <nil>
--------------------
{"Kind":0,"Label":47,"Prefix":"/users/","Parent":null,"Children":[{}],"ChildrenNum":1,"Ppath":"","Pnames":null,"MethodHandler":{}} <nil>
0
{"Kind":1,"Label":58,"Prefix":":","Parent":{},"Children":[{}],"ChildrenNum":1,"Ppath":"/users/:name","Pnames":["name"],"MethodHandler":{}} <nil>
0
{"Kind":0,"Label":47,"Prefix":"/share/","Parent":{},"Children":[{}],"ChildrenNum":1,"Ppath":"","Pnames":null,"MethodHandler":{}} <nil>
0
{"Kind":1,"Label":58,"Prefix":":","Parent":{},"Children":null,"ChildrenNum":0,"Ppath":"/users/:name/share/:id","Pnames":["name","id"],"MethodHandler":{}} <nil>
--------------------
{"Kind":0,"Label":47,"Prefix":"/users/","Parent":null,"Children":[{},{}],"ChildrenNum":2,"Ppath":"","Pnames":null,"MethodHandler":{}} <nil>
0
{"Kind":1,"Label":58,"Prefix":":","Parent":{},"Children":[{}],"ChildrenNum":1,"Ppath":"/users/:name","Pnames":["name"],"MethodHandler":{}} <nil>
0
{"Kind":0,"Label":47,"Prefix":"/share/","Parent":{},"Children":[{}],"ChildrenNum":1,"Ppath":"","Pnames":null,"MethodHandler":{}} <nil>
0
{"Kind":1,"Label":58,"Prefix":":","Parent":{},"Children":null,"ChildrenNum":0,"Ppath":"/users/:name/share/:id","Pnames":["name","id"],"MethodHandler":{}} <nil>
1
{"Kind":0,"Label":110,"Prefix":"names","Parent":{},"Children":null,"ChildrenNum":0,"Ppath":"/users/names","Pnames":[],"MethodHandler":{}} <nil>
--------------------
{"Kind":0,"Label":47,"Prefix":"/users/","Parent":null,"Children":[{},{}],"ChildrenNum":2,"Ppath":"","Pnames":null,"MethodHandler":{}} <nil>
0
{"Kind":1,"Label":58,"Prefix":":","Parent":{},"Children":[{}],"ChildrenNum":1,"Ppath":"/users/:name","Pnames":["name"],"MethodHandler":{}} <nil>
0
{"Kind":0,"Label":47,"Prefix":"/share/","Parent":{},"Children":[{}],"ChildrenNum":1,"Ppath":"","Pnames":null,"MethodHandler":{}} <nil>
0
{"Kind":1,"Label":58,"Prefix":":","Parent":{},"Children":null,"ChildrenNum":0,"Ppath":"/users/:name/share/:id","Pnames":["name","id"],"MethodHandler":{}} <nil>
1
{"Kind":0,"Label":110,"Prefix":"names","Parent":{},"Children":[{}],"ChildrenNum":1,"Ppath":"/users/names","Pnames":[],"MethodHandler":{}} <nil>
0
{"Kind":0,"Label":47,"Prefix":"/","Parent":{},"Children":[{}],"ChildrenNum":1,"Ppath":"","Pnames":null,"MethodHandler":{}} <nil>
0
{"Kind":2,"Label":42,"Prefix":"*","Parent":{},"Children":null,"ChildrenNum":0,"Ppath":"/users/names/*","Pnames":["*"],"MethodHandler":{}} <nil>

func (e *Echo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.router.Find(r.Method, getPath(r), c)
}


func (r *Router) Find(method, path string, c Context) {
	v, err := json.Marshal(struct {
		//Request *http.Request
		//Response *Response
		Path    string
		Pnames  []string
		Pvalues []string
		Query   url.Values
		//Handler HandlerFunc
		Store Map
	}{
		//Request: ctx.request,
		//Response: ctx.response,
		Path:    ctx.path,
		Pnames:  ctx.pnames,
		Pvalues: ctx.pvalues,
		Query:   ctx.query,
		//Handler: ctx.handler,
		Store: ctx.store,
	})
	fmt.Println(string(v), err)
	return
}

//{"Path":"/users/:name","Pnames":["name"],"Pvalues":["Joe",""],"Query":null,"Store":null} <nil>
//{"Path":"/users/:name/share/:id","Pnames":["name","id"],"Pvalues":["Joe","1"],"Query":null,"Store":null} <nil>


% curl -XGET http://localhost:1336/users/Joe/share\?email\=joe_email
{"message":"Not Found"}
% curl -XGET http://localhost:1336/users/Joe/share/1\?email\=joe_email
{"name":"Joe","email":"joe_email"}


func (b *DefaultBinder) Bind(i interface{}, c Context) (err error) {
	if req.ContentLength == 0 {
        if err = b.bindData(i, c.QueryParams(), "query"); err != nil {
        }
    }

    ctype := req.Header.Get(HeaderContentType)
	switch {
	case strings.HasPrefix(ctype, MIMEApplicationJSON):
		if err = json.NewDecoder(req.Body).Decode(i); err != nil {

        }
    }
}

 % curl -i -XGET http://localhost:1336/users/Joe/share/1\?email\=joe_email
HTTP/1.1 200 OK
Content-Type: application/json; charset=UTF-8
Date: Tue, 30 Mar 2021 03:40:22 GMT
Content-Length: 35
{"name":"","email":"joe_email"}



func (b *DefaultBinder) bindData(ptr interface{}, data map[string][]string, tag string) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()
    for i := 0; i < typ.NumField(); i++ {
        inputFieldName := typeField.Tag.Get(tag)

        			// If tag is nil, we inspect if the field is a struct.
			if _, ok := bindUnmarshaler(structField); !ok && structFieldKind == reflect.Struct {
				if err := b.bindData(structField.Addr().Interface(), data, tag); err != nil {
                }
            }

            inputValue, exists := data[inputFieldName]

    }
}

QueryParams() url.Values


 % go get -u github.com/labstack/echo/v4@v4.1.17
go: finding module for package github.com/labstack/echo
代码里引用的地方也要由
	"github.com/labstack/echo"
    改成
    	"github.com/labstack/echo/v4"
        否则会
go: found github.com/labstack/echo in github.com/labstack/echo v3.3.10+incompatible

https://github.com/golang/go/issues/34330
https://my.oschina.net/renhc/blog/3167195
https://www.cnblogs.com/apocelipes/p/10295096.html
https://stackoverflow.com/questions/57355929/what-does-incompatible-in-go-mod-mean-will-it-cause-harm

% go get -u github.com/labstack/echo/v4@v4.1.17

 % curl -i -XGET http://localhost:1336/users/Joe/share/1\?email\=joe_email
HTTP/1.1 200 OK
Content-Type: application/json; charset=UTF-8
Date: Tue, 30 Mar 2021 05:21:10 GMT
Content-Length: 35

{"name":"Joe","email":"joe_email"}



// Bind implements the `Binder#Bind` function.
func (b *DefaultBinder) Bind(i interface{}, c Context) (err error) {
	req := c.Request()

	names := c.ParamNames()
	values := c.ParamValues()
	params := map[string][]string{}
	for i, name := range names {
		params[name] = []string{values[i]}
	}
	if err := b.bindData(i, params, "param"); err != nil {
		return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	if err = b.bindData(i, c.QueryParams(), "query"); err != nil {
		return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	if req.ContentLength == 0 {
		return
	}
	ctype := req.Header.Get(HeaderContentType)
	switch {
	case strings.HasPrefix(ctype, MIMEApplicationJSON):
		if err = json.NewDecoder(req.Body).Decode(i); err != nil {
			if ute, ok := err.(*json.UnmarshalTypeError); ok {
				return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
			} else if se, ok := err.(*json.SyntaxError); ok {
				return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
			}
			return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
	}
	return
}

 curl -i -XGET http://localhost:1336/users/Joe/share/1\?email\=joe_email
HTTP/1.1 200 OK
Content-Type: application/json; charset=UTF-8
Date: Tue, 30 Mar 2021 04:59:50 GMT
Content-Length: 32

{"name":"Joe","email":"joe_email"}

func (b *DefaultBinder) bindData(ptr interface{}, data map[string][]string, tag string) error {
	if ptr == nil || len(data) == 0 {
		return nil
	}
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	// Map
	if typ.Kind() == reflect.Map {
		for k, v := range data {
			val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
		}
		return nil
	}

	// !struct
	if typ.Kind() != reflect.Struct {
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)

		if inputFieldName == "" {
			inputFieldName = typeField.Name //在4.2.1中删除了这个字段
			// If tag is nil, we inspect if the field is a struct.
			if _, ok := structField.Addr().Interface().(BindUnmarshaler); !ok && structFieldKind == reflect.Struct {
				if err := b.bindData(structField.Addr().Interface(), data, tag); err != nil {
					return err
				}
				continue
			}
		}

		inputValue, exists := data[inputFieldName]
		if !exists {
			// Go json.Unmarshal supports case insensitive binding.  However the
			// url params are bound case sensitive which is inconsistent.  To
			// fix this we must check all of the map values in a
			// case-insensitive search.
			for k, v := range data {
				if strings.EqualFold(k, inputFieldName) {
					inputValue = v
					exists = true
					break
				}
			}
		}

		if !exists {
			continue
		}

		// Call this first, in case we're dealing with an alias to an array type
		if ok, err := unmarshalField(typeField.Type.Kind(), inputValue[0], structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		numElems := len(inputValue)
		if structFieldKind == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for j := 0; j < numElems; j++ {
				if err := setWithProperType(sliceOf, inputValue[j], slice.Index(j)); err != nil {
					return err
				}
			}
			val.Field(i).Set(slice)
		} else if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
			return err

		}
	}
	return nil
}




 % go get -u github.com/labstack/echo/v4

 func (b *DefaultBinder) Bind(i interface{}, c Context) (err error) {
	if err := b.BindPathParams(c, i); err != nil {
		return err
	}
    	if c.Request().Method == http.MethodGet || c.Request().Method == http.MethodDelete {
		if err = b.BindQueryParams(c, i); err != nil {
			return err
		}
	}
	return b.BindBody(c, i)
}


func (b *DefaultBinder) BindPathParams(c Context, i interface{}) error {
	names := c.ParamNames()
	values := c.ParamValues()
	params := map[string][]string{}
	for i, name := range names {
		params[name] = []string{values[i]}
	}
	if err := b.bindData(i, params, "param"); err != nil {
		return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	return nil
}


func (b *DefaultBinder) BindQueryParams(c Context, i interface{}) error {
	if err := b.bindData(i, c.QueryParams(), "query"); err != nil {
		return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	return nil
}


func (b *DefaultBinder) BindBody(c Context, i interface{}) (err error) {
	req := c.Request()
	if req.ContentLength == 0 {
		return
	}

	ctype := req.Header.Get(HeaderContentType)
	switch {
	case strings.HasPrefix(ctype, MIMEApplicationJSON):
		if err = json.NewDecoder(req.Body).Decode(i); err != nil {
        }
    }
}


func (b *DefaultBinder) bindData(destination interface{}, data map[string][]string, tag string) error {
	if destination == nil || len(data) == 0 {
		return nil
	}
	typ := reflect.TypeOf(destination).Elem()
	val := reflect.ValueOf(destination).Elem()

	// Map
	if typ.Kind() == reflect.Map {
		for k, v := range data {
			val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
		}
		return nil
	}

	// !struct
	if typ.Kind() != reflect.Struct {
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)

		if inputFieldName == "" {
			// If tag is nil, we inspect if the field is a not BindUnmarshaler struct and try to bind data into it (might contains fields with tags).
			// structs that implement BindUnmarshaler are binded only when they have explicit tag
			if _, ok := structField.Addr().Interface().(BindUnmarshaler); !ok && structFieldKind == reflect.Struct {
				if err := b.bindData(structField.Addr().Interface(), data, tag); err != nil {
					return err
				}
			}
			// does not have explicit tag and is not an ordinary struct - so move to next field
			continue  //注意从哪部移动出来了，所以，没有tag就不继续了
		}

		inputValue, exists := data[inputFieldName]
		if !exists {
			// Go json.Unmarshal supports case insensitive binding.  However the
			// url params are bound case sensitive which is inconsistent.  To
			// fix this we must check all of the map values in a
			// case-insensitive search.
			for k, v := range data {
				if strings.EqualFold(k, inputFieldName) {
					inputValue = v
					exists = true
					break
				}
			}
		}

		if !exists {
			continue
		}

		// Call this first, in case we're dealing with an alias to an array type
		if ok, err := unmarshalField(typeField.Type.Kind(), inputValue[0], structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		numElems := len(inputValue)
		if structFieldKind == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for j := 0; j < numElems; j++ {
				if err := setWithProperType(sliceOf, inputValue[j], slice.Index(j)); err != nil {
					return err
				}
			}
			val.Field(i).Set(slice)
		} else if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
			return err

		}
	}
	return nil
}


 % curl -i -XGET http://localhost:1336/users/Joe/share/1\?email\=joe_email
HTTP/1.1 200 OK
Content-Type: application/json; charset=UTF-8
Date: Tue, 30 Mar 2021 05:22:30 GMT
Content-Length: 35

{"name":"Joe","email":"joe_email"}


Name  string `json:"name" xml:"name` //param:"name" query:"name" form:"name" 

 % curl -i -XGET http://localhost:1336/users/Joe/share/1\?email\=joe_email
HTTP/1.1 200 OK
Content-Type: application/json; charset=UTF-8
Date: Tue, 30 Mar 2021 05:23:30 GMT
Content-Length: 32

{"name":"","email":"joe_email"}



% vimdiff ~/go/pkg/mod/github.com/labstack/echo/v4@v4.1.17/bind.go ~/go/pkg/mod/github.com/labstack/echo/v4@v4.2.1/bind.go 