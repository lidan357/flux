package filter

import (
	"fmt"
	"github.com/bytepowered/flux"
	"github.com/bytepowered/flux/extension"
	"github.com/bytepowered/flux/logger"
	"github.com/bytepowered/flux/pkg"
	"github.com/bytepowered/lakego"
	"strings"
	"time"
)

const (
	TypeNameFilterPermissionVerification = "PermissionVerification"
)

var (
	ErrPermissionSubjectNotFound = &flux.InvokeError{
		StatusCode: flux.StatusBadRequest,
		Message:    "PERMISSION:SUBJECT_NOT_FOUND",
	}
	ErrPermissionSubjectAccessDenied = &flux.InvokeError{
		StatusCode: flux.StatusAccessDenied,
		Message:    "PERMISSION:SUBJECT_ACCESS_DENIED",
	}
)

type PermissionProvider func(subjectId, method, uri string) (bool, error)

func PermissionVerificationFactory() interface{} {
	return new(permissionFilter)
}

func NewPermissionVerificationWith(provider PermissionProvider) flux.Filter {
	return &permissionFilter{
		provider: provider,
	}
}

// Permission Filter，负责读取JWT的Subject字段，调用指定Dubbo接口判断权限
type permissionFilter struct {
	disabled  bool
	provider  PermissionProvider
	permCache lakego.Cache
}

func (p *permissionFilter) Invoke(next flux.FilterInvoker) flux.FilterInvoker {
	if p.disabled {
		return next
	}
	return func(ctx flux.Context) *flux.InvokeError {
		if false == ctx.Endpoint().Authorize {
			return next(ctx)
		}
		if err := p.verify(ctx); nil != err {
			return err
		} else {
			return next(ctx)
		}
	}
}

func (p *permissionFilter) Init(config flux.Config) error {
	p.disabled = config.BooleanOrDefault(keyConfigDisabled, false)
	if p.disabled {
		logger.Infof("Permission filter was DISABLED!!")
		return nil
	}
	if !config.IsEmpty() && p.provider == nil {
		p.provider = func() PermissionProvider {
			proto := config.String(keyConfigVerificationProtocol)
			upsHost := config.String(keyConfigVerificationHost)
			upsUri := config.String(keyConfigVerificationUri)
			upsMethod := config.String(keyConfigVerificationMethod)
			logger.Infof("Permission filter config provider, proto:%s, method: %s, uri: %s%s", proto, upsMethod, upsHost, upsHost)
			return func(subjectId, method, pattern string) (bool, error) {
				switch strings.ToUpper(proto) {
				case flux.ProtocolDubbo:
					return p.loadByExchange(flux.ProtocolDubbo, upsHost, upsMethod, upsUri, subjectId, method, pattern)
				case flux.ProtocolHttp:
					return p.loadByExchange(flux.ProtocolHttp, upsHost, upsMethod, upsUri, subjectId, method, pattern)
				default:
					return false, fmt.Errorf("unknown verification protocol: %s", proto)
				}
			}
		}()
	}
	logger.Infof("Permission filter initializing, config: %+v", config)
	// TODO 检查参数
	permCacheExpiration := config.Int64OrDefault(keyConfigCacheExpiration, defValueCacheExpiration)
	p.permCache = lakego.NewSimple(lakego.WithExpiration(time.Minute * time.Duration(permCacheExpiration)))
	return nil
}

func (p *permissionFilter) Order() int {
	return OrderFilterPermissionVerification
}

func (p *permissionFilter) verify(ctx flux.Context) *flux.InvokeError {
	jwtSubjectId, ok := ctx.AttrValue(flux.XJwtSubject)
	if !ok {
		return ErrPermissionSubjectNotFound
	}
	endpoint := ctx.Endpoint()
	// 验证用户是否有权限访问API：(userSubId, method, uri-pattern)
	permKey := fmt.Sprintf("%s@%s#%s", jwtSubjectId, endpoint.HttpMethod, endpoint.HttpPattern)
	allowed, err := p.permCache.GetOrLoad(permKey, func(_ lakego.Key) (lakego.Value, error) {
		strSubId := pkg.ToString(jwtSubjectId)
		return p.provider(strSubId, endpoint.HttpMethod, endpoint.HttpPattern)
	})
	if err == nil {
		if v, ok := allowed.(bool); ok && v {
			return nil
		} else {
			return ErrPermissionSubjectAccessDenied
		}
	} else {
		return &flux.InvokeError{
			StatusCode: flux.StatusServerError,
			Message:    "PERMISSION:LOAD_ACCESS",
			Internal:   err,
		}
	}
}

func (p *permissionFilter) loadByExchange(proto string,
	upsHost, upsMethod, upsUri string,
	reqSubjectId, reqMethod, reqPattern string) (bool, error) {
	exchange, _ := extension.GetExchange(proto)
	if ret, err := exchange.Invoke(&flux.Endpoint{
		UpstreamHost:   upsHost,
		UpstreamMethod: upsMethod,
		UpstreamUri:    upsUri,
		Arguments: []flux.Argument{
			{TypeClass: pkg.JavaLangStringClassName, ArgName: "subjectId", ArgValue: flux.NewWrapValue(reqSubjectId)},
			{TypeClass: pkg.JavaLangStringClassName, ArgName: "method", ArgValue: flux.NewWrapValue(reqMethod)},
			{TypeClass: pkg.JavaLangStringClassName, ArgName: "pattern", ArgValue: flux.NewWrapValue(reqPattern)},
		},
	}, nil); nil != err {
		return false, err
	} else {
		return strings.Contains(pkg.ToString(ret), "success"), nil
	}
}
