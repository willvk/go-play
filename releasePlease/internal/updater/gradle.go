package updater

import "regexp"

var (
	propsVersionRe  = regexp.MustCompile(`(?m)^([ \t]*version[ \t]*[=:][ \t]*)\S+`)
	gradleVersionRe = regexp.MustCompile(`(?m)^([ \t]*(?:project\.)?version[ \t]*=[ \t]*['"])[^'"\n]*(['"])`)
)

// GradleProperties replaces the value of the `version` key, e.g.
// `version=1.2.3-SNAPSHOT` becomes `version=1.3.0`.
func GradleProperties(content, version string) string {
	return propsVersionRe.ReplaceAllString(content, "${1}"+version)
}

// GradleBuild replaces `version = '...'` / `version = "..."` assignments in
// build.gradle (Groovy) and build.gradle.kts (Kotlin) scripts.
func GradleBuild(content, version string) string {
	return gradleVersionRe.ReplaceAllString(content, "${1}"+version+"${2}")
}
