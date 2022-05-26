package cmd

import (
	"testing"
)

func TestFindDOI(t *testing.T) {
	table := map[string]string{
		``:                       ``,
		`-`:                      ``,
		`10.1093/mnras/staa3102`: `10.1093/mnras/staa3102`,
		`https://doi.org/10.1016/j.jde.2019.11.028`:  `10.1016/j.jde.2019.11.028`,
		`DOI: 10.1080/09518398.2019.1678783`:         `10.1080/09518398.2019.1678783`,
		`https://doi.org/10.1177%2F0276146720949636`: `10.1177/0276146720949636`,
		` 10.1128/msphere.00864-20 `:                 `10.1128/msphere.00864-20`,
		`10.33423/ajm.v21i4.4555`:                    `10.33423/ajm.v21i4.4555`,
		`10.1007/978-3-030-40274-7_89`:               `10.1007/978-3-030-40274-7_89`,
	}
	for in, expect := range table {
		if out := cleanDOI(in); out != expect {
			t.Errorf(`for %s, expected %s, got: %s`, in, expect, out)
		}
	}

}

func TestResolveDOI(t *testing.T) {
	table := map[string]bool{
		`10.1093/mnras/staa3102`:                    true,
		`https://doi.org/10.1016/j.jde.2019.11.028`: true,
		`DOI: 10.1080/09518398.2019.1678783`:        true,
		`asdf`:                                      false,
		`10.1128/jokejokesjokes`:                    false,
		``:                                          false,
		`10.1007/978-3-030-40274-7_89`:              true,
	}
	for in, expect := range table {
		if out := resolvableDOI(in); out != expect {
			t.Errorf(`for %s, expected %v, got %v`, in, expect, out)
		}
	}

}
