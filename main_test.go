package main

import "testing"

func TestExtractStructs(t *testing.T) {
	code := `
struct SceneSample {
    float closest_distance;
    int index;
};

struct RayEnd {
    SceneSample s;
    vec3 current_position;
};
		
	`
	structs, err := extractStructs(code)
	if len(structs) != 2 {
		t.Errorf("Unexpected amount of structs %d", len(structs))
	}
	if err != nil {
		t.Errorf("extracting structs failed with error %#v\n", err)
	}

	if structs[0].Name != "SceneSample" {
		t.Errorf("Unexpected name for %#v\n", structs[0])
	}
	if structs[1].Name != "RayEnd" {
		t.Errorf("Unexpected name for %#v\n", structs[1])
	}

	if structs[0].Range.Start.Line != 1 {
		t.Errorf("Unexpected line start for %#v", structs[0])
	}
	if structs[0].Range.End.Line != 1 {
		t.Errorf("Unexpected line end for %#v", structs[0])
	}

	if structs[0].Range.Start.Character != 7 {
		t.Errorf("Unexpected char start for %#v", structs[0])
	}
	if structs[0].Range.End.Character != 17 {
		t.Errorf("Unexpected char end for %#v", structs[0])
	}
}

func TestExtractFunctions(t *testing.T) {
	code := `
struct SceneSample {
    float closest_distance;
    int index;
};

struct RayEnd {
    SceneSample s;
    vec3 current_position;
};

RayEnd g();
		
	`
	structs, err := extractStructs(code)
	if err != nil {
		t.Errorf("extracting structs failed with error %#v\n", err)
	}

	functions, err := extractFunctions(code, structs)

	if len(functions) != 1 {
		t.Errorf("Failed to extract functions %#v", functions)
	}
	if functions[0].Name != "g" {

		t.Errorf("Failed to extract function name %#v", functions)
	}
}
