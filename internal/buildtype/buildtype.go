package buildtype

type BuildType string

const (
	DEBUG   BuildType = "debug"
	RELEASE BuildType = "release"
)

func (t BuildType) Valid() bool {
	return t == DEBUG || t == RELEASE
}

func (t BuildType) Resolved(fallback string) BuildType {
	if t.Valid() {
		return t
	}
	if BuildType(fallback).Valid() {
		return BuildType(fallback)
	}
	return DEBUG
}
