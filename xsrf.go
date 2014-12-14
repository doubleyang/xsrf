package xsrf

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/lunny/tango"
	"github.com/go-xweb/uuid"
)

const (
	XSRF_TAG string = "_xsrf"
)

func XsrfName() string {
	return XSRF_TAG
}

type XsrfOptionInterface interface {
	CheckXsrf() bool
}

type Xsrf struct {
	render *tango.Render
	timeout time.Duration
}

func NewXsrf(timeout time.Duration) *Xsrf {
	return &Xsrf{
		timeout:timeout,
	}
}

func (xsrf *Xsrf) SetRender(render *tango.Render) {
	xsrf.render = render
	render.FuncMaps["XsrfName"] = XsrfName
}

// NewCookie is a helper method that returns a new http.Cookie object.
// Duration is specified in seconds. If the duration is zero, the cookie is permanent.
// This can be used in conjunction with ctx.SetCookie.
func newCookie(name string, value string, age int64) *http.Cookie {
	var utctime time.Time
	if age == 0 {
		// 2^31 - 1 seconds (roughly 2038)
		utctime = time.Unix(2147483647, 0)
	} else {
		utctime = time.Unix(time.Now().Unix()+age, 0)
	}
	return &http.Cookie{Name: name, Value: value, Expires: utctime}
}

func (xsrf *Xsrf) Handle(ctx *tango.Context) {
	var action interface{}
	if action = ctx.Action(); action == nil {
		ctx.Next()
		return
	}

	// if action implements check xsrf option and ask not check then return
	if checker, ok := action.(XsrfOptionInterface); ok && !checker.CheckXsrf() {
		ctx.Next()
		return
	}

	if ctx.Req().Method == "GET" {
		xsrf.render.FuncMaps["XsrfName"] = XSRF_TAG

		var val string = ""
		cookie, err := ctx.Req().Cookie(XSRF_TAG)
		if err != nil {
			val = uuid.NewRandom().String()
			cookie = newCookie(XSRF_TAG, val, int64(xsrf.timeout))
			ctx.SetHeader("Set-Cookie", cookie.String())
		} else {
			val = cookie.Value
		}

		xsrf.render.FuncMaps["XsrfValue"] = func() string {
			return val
		}
		xsrf.render.FuncMaps["XsrfFormHtml"] = func() template.HTML {
			return template.HTML(fmt.Sprintf(`<input type="hidden" name="%v" value="%v" />`,
			XSRF_TAG, val))
		}
	} else if ctx.Req().Method == "POST" {
		res, err := ctx.Req().Cookie(XSRF_TAG)
		formVals := ctx.Req().Form[XSRF_TAG]
		var formVal string
		if len(formVals) > 0 {
			formVal = formVals[0]
		}
		if err != nil || res.Value == "" || res.Value != formVal {
			ctx.WriteHeader(http.StatusInternalServerError)
			ctx.Write([]byte("xsrf token error."))
			return
		}
	}

	ctx.Next()
}