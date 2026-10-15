package httpserve

type tlsCert struct {
	key  string
	cert string
}

func (t *tlsCert) isValid() (ok bool) {
	if len(t.key) == 0 {
		return
	}

	if len(t.cert) == 0 {
		return
	}

	return true
}
