package fhirgen

import (
	"encoding/json"
	"fmt"
	"log"

	fhir "github.com/jlcoulter/fhir-registry"
)

// exampleRegistry loads the sample package used across the examples. It is
// separate from loadTestRegistry (which needs a *testing.T) so that Example
// functions, which have no T, can share it.
func exampleRegistry() *fhir.Registry {
	reg := fhir.NewRegistry()
	if err := reg.LoadPackageTgz("testdata/au-base.tgz"); err != nil {
		log.Fatal(err)
	}
	return reg
}

// ExampleNew demonstrates constructing a Generator from a registry.
func ExampleNew() {
	reg := exampleRegistry()
	g := New(reg)
	_ = g
}

// ExampleGenerator_Generate demonstrates generating a resource by base type
// name in the default minimal-fill mode (only required elements).
func ExampleGenerator_Generate() {
	reg := exampleRegistry()
	g := New(reg, WithSeed(42))

	patient, err := g.Generate("Patient")
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(patient, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

// ExampleGenerator_Generate_fullMode demonstrates generating a resource with
// every element filled, including optional ones.
func ExampleGenerator_Generate_fullMode() {
	reg := exampleRegistry()
	g := New(reg, WithSeed(42), WithFullFillMode())

	org, err := g.Generate("Organization")
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(org, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

// ExampleGenerator_GenerateForURL demonstrates generating a resource for a
// specific StructureDefinition canonical URL, which honors that profile.
func ExampleGenerator_GenerateForURL() {
	reg := exampleRegistry()
	g := New(reg, WithSeed(42))

	patient, err := g.GenerateForURL("http://hl7.org.au/fhir/StructureDefinition/au-patient")
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(patient, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

// ExampleWithSeed demonstrates that two generators sharing a seed produce
// identical output.
func ExampleWithSeed() {
	reg := exampleRegistry()
	g1 := New(reg, WithSeed(7))
	g2 := New(reg, WithSeed(7))

	a, err := g1.Generate("Patient")
	if err != nil {
		log.Fatal(err)
	}
	b, err := g2.Generate("Patient")
	if err != nil {
		log.Fatal(err)
	}

	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	fmt.Println(string(aj) == string(bj))
}

// ExampleWithValues demonstrates overriding specific fields by FHIR element
// path. Paths are relative to the resource root; unset fields are still
// generated with fake data.
func ExampleWithValues() {
	reg := exampleRegistry()
	g := New(reg, WithSeed(42), WithValues(map[string]any{
		"birthDate":    "1990-01-01",
		"address.city": "Sydney",
	}))

	patient, err := g.Generate("Patient")
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(patient, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}
