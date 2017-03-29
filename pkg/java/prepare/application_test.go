package prepare_test

import (
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestClasspathOrder(t *testing.T) {

	// Given
	root := setupApplication(t)

	// When
	actualCp, err := prepare.Classpath(filepath.Join(root, "lib"))

	if err != nil {
		t.Fatal("Failed to test classpath: ", err)
	}

	// Then
	for idx, entry := range []string{"bar.jar", "bar2.jar", "foo.jar", "foobar.jar"} {
		expectedLib := filepath.Join(root, "lib", entry)

		if idx >= len(actualCp) {
			t.Error("Classpath", actualCp, "is not complete")
			return
		}

		actualLib := actualCp[idx]
		if actualLib != expectedLib {
			t.Error("Classpath", actualCp, "is not correct, excpected", expectedLib, ", got", actualLib)
		}
	}

	deleteApplication(root)
}

func TestPrepareStartscript(t *testing.T) {

	// Given
	root := setupApplication(t)

	// When
	prepare.PrepareApplication(root, TestMeta)

	// Then
	scriptExists, err := prepare.Exists(filepath.Join(root, "bin", "generated-start"))

	if err != nil || !scriptExists {
		t.Error("Failed to generate startscript")
	}

	linkExists, err := prepare.Exists(filepath.Join(root, "bin", "os-start"))

	if err != nil || !linkExists {
		t.Error("Failed to generate link to startscript")
	}

	//deleteApplication(root)
}

func setupApplication(t *testing.T) string {

	// Base folder
	root, err := ioutil.TempDir("", "architect-test")

	if err != nil {
		t.Fatal("Failed to create test application: ", err)
	}

	// Sub folders
	for _, folder := range []string{"lib", "bin"} {
		err = os.MkdirAll(filepath.Join(root, folder), 0766)

		if err != nil {
			deleteApplication(root)
			t.Fatal("Failed to create test application: ", err)
		}
	}

	// Libraries
	for _, lib := range []string{"foo.jar", "bar.jar", "foobar.jar", "bar2.jar", "foo.jar"} {
		err := ioutil.WriteFile(filepath.Join(root, "lib", lib), []byte("FOO"), 0666)

		if err != nil {
			deleteApplication(root)
			t.Fatal("Failed to create test application: ", err)
		}
	}

	return root
}

func deleteApplication(root string) error {
	return os.RemoveAll(root)
}