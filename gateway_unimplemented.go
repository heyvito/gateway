//go:build !(darwin || linux)

package gateway

func init() {
	getRoutes = func() (NetRouteList, error) {
		return nil, ErrNotImplemented{}
	}
}
