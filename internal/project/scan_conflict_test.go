package project

import "testing"

func androidRequirements(module Module) []RuntimeRequirement {
	var out []RuntimeRequirement
	for _, requirement := range module.Requirements {
		if requirement.Runtime == "androidsdk" {
			out = append(out, requirement)
		}
	}
	return out
}

func javaRequirements(module Module) []RuntimeRequirement {
	var out []RuntimeRequirement
	for _, requirement := range module.Requirements {
		if requirement.Runtime == "java" {
			out = append(out, requirement)
		}
	}
	return out
}

func statuses(requirements []RuntimeRequirement) []RequirementStatus {
	out := make([]RequirementStatus, 0, len(requirements))
	for _, item := range requirements {
		out = append(out, item.Status)
	}
	return out
}

// TestAndroidCompileAndMinSdkAreNotConflicts：compileSdk 与 minSdk 是
// 同一运行时的两个属性，必须各自 resolved，不得误报 conflict（图 4 用例）。
func TestAndroidCompileAndMinSdkAreNotConflicts(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"build.gradle": "android {\n    compileSdk 31\n    defaultConfig {\n        minSdk 21\n    }\n}\n",
	}))
	requirements := androidRequirements(module)
	if len(requirements) != 2 {
		t.Fatalf("expected two androidsdk properties, got %+v", requirements)
	}
	for _, item := range requirements {
		if item.Status != RequirementResolved {
			t.Fatalf("compileSdk/minSdk must be resolved, got %+v", item)
		}
	}
	if requirements[0].RawConstraint != "31" || requirements[1].RawConstraint != "21" {
		t.Fatalf("stable order expected (compileSdk then minSdk): %+v", requirements)
	}
}

// TestSamePropertyDifferentValuesConflict：同一属性多个互斥精确版本
// 才是真冲突（两个 compileSdk 声明）。
func TestSamePropertyDifferentValuesConflict(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"build.gradle": "android {\n    compileSdk 31\n}\nandroid {\n    compileSdk 33\n}\n",
	}))
	requirements := androidRequirements(module)
	if len(requirements) != 2 {
		t.Fatalf("expected two compileSdk rows, got %+v", requirements)
	}
	for _, item := range requirements {
		if item.Status != RequirementConflict {
			t.Fatalf("two different compileSdk values must conflict, got %+v", item)
		}
	}
}

// TestMavenCompilerPropertiesDifferentValuesConflict：java 是单一
// 语义属性，maven.compiler.release 与 java.version 给出不同精确值时
// 必须报冲突（同一运行时不并存两个版本）。
func TestMavenCompilerPropertiesDifferentValuesConflict(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"pom.xml": "<project><properties><maven.compiler.release>17</maven.compiler.release><java.version>21</java.version></properties></project>",
	}))
	requirements := javaRequirements(module)
	if len(requirements) != 2 {
		t.Fatalf("expected two java properties, got %+v", requirements)
	}
	for _, item := range requirements {
		if item.Status != RequirementConflict {
			t.Fatalf("two different java versions must conflict, got %+v", item)
		}
	}
}

// TestRangeAndExactMixFallsBackUnknown：区间与精确值混合（非全精确）
// 时保持 unknown，不擅自任选其一。
func TestRangeAndExactMixFallsBackUnknown(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"package.json": "{\"engines\":{\"node\":\">=18\"}}",
		".nvmrc":       "20\n",
	}))
	seenNode := 0
	for _, requirement := range module.Requirements {
		if requirement.Runtime == "node" {
			seenNode++
			if requirement.Status != RequirementUnknown {
				t.Fatalf("range + exact mix must be unknown, got %+v", requirement)
			}
		}
	}
	if seenNode != 2 {
		t.Fatalf("expected two node requirements, got %+v", module.Requirements)
	}
}

// TestSamePropertyExactValuesConflict：gradle toolchain 同属性两个精确
// 版本仍按冲突呈现。
func TestSamePropertyExactValuesConflict(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"build.gradle": "java {\n    toolchain {\n        languageVersion = JavaLanguageVersion.of(17)\n    }\n}\njava {\n    toolchain {\n        languageVersion = JavaLanguageVersion.of(21)\n    }\n}\n",
	}))
	found := javaRequirements(module)
	if len(found) != 2 {
		t.Fatalf("expected two java toolchain rows, got %+v", found)
	}
	for _, item := range statuses(found) {
		if item != RequirementConflict {
			t.Fatalf("same-property exact values must conflict, got %v", item)
		}
	}
}
