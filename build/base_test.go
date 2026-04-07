package build

import (
	. "github.com/aandryashin/matchers"
	"testing"
)

func TestVersionFromPackageName(t *testing.T) {
	AssertThat(t, extractVersion("google-chrome-stable_48.0.2564.109-1+aerokube0_amd64.deb"), EqualTo{"48.0.2564.109"})
	AssertThat(t, extractVersion("google-chrome-beta_79.0.3945.36-1_amd64.deb"), EqualTo{"79.0.3945.36"})
	AssertThat(t, extractVersion("google-chrome-unstable_80.0.3964.0-1_amd64.deb"), EqualTo{"80.0.3964.0"})
}

func TestVersionToTag(t *testing.T) {
	AssertThat(t, extractVersion("45.0.2+build1-0ubuntu0.14.04.1+aerokube0"), EqualTo{"45.0.2"})
	AssertThat(t, extractVersion("48.0.2564.109-1+aerokube0"), EqualTo{"48.0.2564.109"})
	AssertThat(t, extractVersion("78.0.3904.108-1"), EqualTo{"78.0.3904.108"})
	AssertThat(t, extractVersion("38.0.2220.31"), EqualTo{"38.0.2220.31"})
	AssertThat(t, extractVersion("71.0~b11+build1-0ubuntu0.18.04.1"), EqualTo{"71.0"})
	AssertThat(t, extractVersion("74.0~a1~hg20200205r512485-0ubuntu0.18.04.1~umd1"), EqualTo{"74.0"})
}

func TestParseVersion(t *testing.T) {
	AssertThat(t, majorVersion("78.0.3904.108"), EqualTo{"78"})
	AssertThat(t, majorVersion("78"), EqualTo{"78"})
	AssertThat(t, majorMinorVersion("78.0.3904.108"), EqualTo{"78.0"})
	AssertThat(t, majorMinorVersion("78.0"), EqualTo{"78.0"})
	AssertThat(t, buildVersion("78.0.3904.108"), EqualTo{"78.0.3904"})
	AssertThat(t, buildVersion("78.0.3904"), EqualTo{"78.0.3904"})
}
