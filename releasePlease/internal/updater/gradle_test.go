package updater

import "testing"

func TestGradleProperties(t *testing.T) {
	in := "group=com.example\nversion=1.2.3-SNAPSHOT\nversionCode=7\nkotlin_version=1.9.0\n"
	want := "group=com.example\nversion=1.3.0\nversionCode=7\nkotlin_version=1.9.0\n"
	if got := GradleProperties(in, "1.3.0"); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if got := GradleProperties("version: 0.1.0\n", "0.2.0"); got != "version: 0.2.0\n" {
		t.Errorf("colon form: got %q", got)
	}
}

func TestGradleBuild(t *testing.T) {
	groovy := "plugins { id 'java' }\ngroup = 'com.example'\nversion = '1.2.3'\n\ndependencies {\n    implementation 'x:y:1.0.0'\n}\n"
	if got := GradleBuild(groovy, "2.0.0"); got != "plugins { id 'java' }\ngroup = 'com.example'\nversion = '2.0.0'\n\ndependencies {\n    implementation 'x:y:1.0.0'\n}\n" {
		t.Errorf("groovy: got\n%s", got)
	}
	kts := "allprojects {\n    project.version = \"1.2.3-SNAPSHOT\"\n}\nval versionName = \"9.9.9\"\n"
	if got := GradleBuild(kts, "1.3.0"); got != "allprojects {\n    project.version = \"1.3.0\"\n}\nval versionName = \"9.9.9\"\n" {
		t.Errorf("kts: got\n%s", got)
	}
}
