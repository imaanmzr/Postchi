package detect

import "testing"

func TestDetectByExtensionEmptyContent(t *testing.T) {
	cases := []struct {
		filename string
		want     Format
	}{
		{filename: "spec.yaml", want: FormatOpenCollection},
		{filename: "spec.yml", want: FormatOpenCollection},
		{filename: "api.json", want: FormatPostman},
		{filename: "readme.txt", want: FormatUnknown},
		{filename: "archive.zip", want: FormatBruno},
	}
	for _, tc := range cases {
		if got := DetectFormat(tc.filename, nil); got != tc.want {
			t.Fatalf("%s => %q want %q", tc.filename, got, tc.want)
		}
	}
}

func TestMatchYAMLMarkerSkipsComments(t *testing.T) {
	content := "# comment\n\nopenapi: 3.0.0\n"
	if got := DetectFormat("spec.yaml", []byte(content)); got != FormatOpenAPI {
		t.Fatalf("got %q", got)
	}
}
