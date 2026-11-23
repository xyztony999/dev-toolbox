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

// TestGradleSourceTargetCompatibilityDeclared：Android 工程最常见的
// sourceCompatibility/targetCompatibility 写法必须识别为 java 已声明
// （SEG2105F21Project_Group_9 报告的"未声明"即此）。
func TestGradleSourceTargetCompatibilityDeclared(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"build.gradle": "android {\n    compileSdk 31\n    defaultConfig {\n        minSdk 21\n    }\n    compileOptions {\n        sourceCompatibility JavaVersion.VERSION_11\n        targetCompatibility JavaVersion.VERSION_11\n    }\n}\n",
	}))
	java := javaRequirements(module)
	if len(java) != 2 {
		t.Fatalf("expected source+target java rows, got %+v", java)
	}
	for _, item := range java {
		if item.Status != RequirementResolved || item.RawConstraint != "11" {
			t.Fatalf("VERSION_11 pair must resolve as 11, got %+v", item)
		}
		if item.Evidence[0].RawValue != "11" || item.Evidence[0].Locator != "sourceCompatibility" && item.Evidence[0].Locator != "targetCompatibility" {
			t.Fatalf("evidence must carry the raw token and property locator, got %+v", item.Evidence[0])
		}
	}
	for _, item := range androidRequirements(module) {
		if item.Status != RequirementResolved {
			t.Fatalf("androidsdk must stay resolved, got %+v", item)
		}
	}
}

// TestGradleSourceTargetDifferentValuesNotConflict：source 11 + target 17
// 是合法共存（字节码兼容属性对），不得报冲突。
func TestGradleSourceTargetDifferentValuesNotConflict(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"build.gradle": "compileOptions {\n    sourceCompatibility JavaVersion.VERSION_11\n    targetCompatibility JavaVersion.VERSION_17\n}\n",
	}))
	java := javaRequirements(module)
	if len(java) != 2 {
		t.Fatalf("expected two rows, got %+v", java)
	}
	for _, item := range java {
		if item.Status != RequirementResolved {
			t.Fatalf("source 11 + target 17 must coexist as resolved, got %+v", item)
		}
	}
}

// TestGradleSourceCompatibilityDuplicateConflict：同一 sourceCompatibility
// 属性的两个不同精确值仍冲突。
func TestGradleSourceCompatibilityDuplicateConflict(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"build.gradle": "sourceCompatibility JavaVersion.VERSION_11\nsourceCompatibility JavaVersion.VERSION_17\n",
	}))
	java := javaRequirements(module)
	if len(java) != 2 {
		t.Fatalf("expected two sourceCompatibility rows, got %+v", java)
	}
	for _, item := range statuses(java) {
		if item != RequirementConflict {
			t.Fatalf("duplicate sourceCompatibility values must conflict, got %v", item)
		}
	}
}

// TestGradleLiteralAndLegacyForms：kts 字面量（"1.8"/"17"）、jvmTarget
// 调用形式与 VERSION_1_8 必须识别；8 与 1.8 同义归一，不互报冲突。
func TestGradleLiteralAndLegacyForms(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"build.gradle.kts": "compileOptions {\n    sourceCompatibility = \"1.8\"\n}\nkotlinOptions {\n    jvmTarget = \"17\"\n}\n",
	}))
	java := javaRequirements(module)
	got := map[string]RuntimeRequirement{}
	for _, item := range java {
		got[item.Evidence[0].Locator] = item
	}
	_, hasSource := got["sourceCompatibility"]
	_, hasJvm := got["jvmTarget"]
	if len(java) != 2 || !hasSource || !hasJvm {
		t.Fatalf("literal 1.8 + jvmTarget 17 must be declared, got %+v", java)
	}
	if got["sourceCompatibility"].RawConstraint != "1.8" || got["jvmTarget"].RawConstraint != "17" {
		t.Fatalf("constraints wrong: %+v", java)
	}
	for _, item := range java {
		if item.Status != RequirementResolved {
			t.Fatalf("literal forms must resolve, got %+v", item)
		}
	}
}

// TestGradleLegacyVersionEquivalence：VERSION_1_8 与字面量 8 同义
// （归一 1.8），同属性同义值不构成冲突。
func TestGradleLegacyVersionEquivalence(t *testing.T) {
	module := makeModule(makeFiles(map[string]string{
		"build.gradle": "sourceCompatibility JavaVersion.VERSION_1_8\ntargetCompatibility = 8\n",
	}))
	java := javaRequirements(module)
	if len(java) != 2 {
		t.Fatalf("expected two rows, got %+v", java)
	}
	for _, item := range java {
		if item.Status != RequirementResolved || item.RawConstraint != "1.8" {
			t.Fatalf("legacy 1_8 and literal 8 must normalize to 1.8 resolved, got %+v", item)
		}
	}
}
