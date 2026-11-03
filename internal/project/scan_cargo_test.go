package project

import "testing"

func makeFiles(files map[string]string) *moduleFiles {
	return &moduleFiles{path: ".", files: files}
}

func hasBuild(module Module, name string) bool {
	for _, build := range module.BuildSystems {
		if build == name {
			return true
		}
	}
	return false
}

func rustRequirement(module Module) *RuntimeRequirement {
	for _, requirement := range module.Requirements {
		if requirement.Runtime == "rust" {
			return &requirement
		}
	}
	return nil
}

func TestCargoModuleExtractsRustVersion(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"cargo.toml": "[package]\nname = \"demo\"\nversion = \"0.1.0\"\nrust-version = \"1.75\"\nedition = \"2021\"\n",
	}))
	if !hasBuild(module, "cargo") {
		t.Fatalf("cargo build system missing: %+v", module.BuildSystems)
	}
	requirement := rustRequirement(module)
	if requirement == nil {
		t.Fatalf("rust requirement missing: %+v", module.Requirements)
	}
	if requirement.RawConstraint != "1.75" || requirement.SourcePath != "cargo.toml" || requirement.Confidence != "high" {
		t.Fatalf("unexpected rust requirement: %+v", requirement)
	}
}

func TestCargoWithoutRustVersionFallsBackToUnknown(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"cargo.toml": "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n",
	}))
	requirement := rustRequirement(module)
	if requirement == nil || requirement.RawConstraint != "" || requirement.Status != RequirementUnknown {
		t.Fatalf("cargo without rust-version must yield unknown requirement: %+v", requirement)
	}
}

func TestRustToolchainPins(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"rust-toolchain.toml": "[toolchain]\nchannel = \"1.82.0\"\nprofile = \"minimal\"\n",
	}))
	if !hasBuild(module, "rust") {
		t.Fatalf("rust build system missing: %+v", module.BuildSystems)
	}
	requirement := rustRequirement(module)
	if requirement == nil || requirement.RawConstraint != "1.82.0" {
		t.Fatalf("toml pin missing: %+v", requirement)
	}

	bare := makeModule(makeFiles(map[string]string{
		"rust-toolchain": "# pinned by CI\nstable\n",
	}))
	bareRequirement := rustRequirement(bare)
	if bareRequirement == nil || bareRequirement.RawConstraint != "stable" {
		t.Fatalf("bare pin missing: %+v", bareRequirement)
	}
}
