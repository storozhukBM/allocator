package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

type allocatorDefinition struct {
	DirName                      string
	PkgName                      string
	TargetTypeName               string
	TypeNameWithUpperFirstLetter string
	Exported                     bool
}

type Generator struct {
	template *template.Template
}

func NewGenerator() *Generator {
	return &Generator{template: template.Must(template.New("embedded").Parse(embeddedTemplate))}
}

// RunGeneratorForTypes generates code for targetTypes into dirName
func (g *Generator) RunGeneratorForTypes(dirName string, targetTypes []string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dirName, nil, parser.SpuriousErrors)
	if err != nil {
		return fmt.Errorf("can't parse destination dir: %v", err)
	}
	var filesToCheck []*ast.File
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			if file != nil {
				filesToCheck = append(filesToCheck, file)
			}
		}
	}

	conf := &types.Config{IgnoreFuncBodies: true, Importer: importer.ForCompiler(fset, "source", nil)}
	typeCheckedPkg, checkErr := conf.Check(dirName, fset, filesToCheck, nil)
	if checkErr != nil {
		return fmt.Errorf("can't check types: %v", checkErr)
	}
	for _, t := range targetTypes {
		obj := typeCheckedPkg.Scope().Lookup(t)
		if obj == nil {
			continue
		}
		generationErr := g.generateAllocators(fset, obj, t)
		if generationErr != nil {
			return fmt.Errorf("can't generate allocator for type: %v: \n%v", obj.Type(), generationErr)
		}
	}
	return nil
}

func (g *Generator) generateAllocators(fset *token.FileSet, obj types.Object, typeName string) error {
	checkPos, checkErr := g.checkObjForInternalPointers(obj, 0)
	if checkErr != nil {
		return fmt.Errorf(
			"target obj '%v' has internal pointers: %v\npointer position: %v\n%v",
			obj.Type(), fset.Position(obj.Pos()), fset.Position(checkPos), checkErr,
		)
	}
	typeNameRunes := bytes.Runes([]byte(typeName))
	typeNameRunes[0] = unicode.ToUpper(typeNameRunes[0])
	typeNameWithUpperFirstLetter := string(typeNameRunes)

	definition := allocatorDefinition{
		DirName:                      obj.Pkg().Path(),
		PkgName:                      obj.Pkg().Name(),
		TargetTypeName:               typeName,
		TypeNameWithUpperFirstLetter: typeNameWithUpperFirstLetter,
		Exported:                     obj.Exported(),
	}
	return g.generateFromTemplateAndWriteToFile(definition)
}

func (g *Generator) generateFromTemplateAndWriteToFile(definition allocatorDefinition) error {
	var b bytes.Buffer
	templateErr := g.template.Execute(&b, definition)
	if templateErr != nil {
		return fmt.Errorf("can't render embedded template: %v", templateErr)
	}
	src, formatErr := format.Source(b.Bytes())
	if formatErr != nil {
		return fmt.Errorf("can't format generated template: %v", formatErr)
	}
	output := strings.ToLower(definition.TargetTypeName + ".alloc.go")
	absPath, pathErr := filepath.Abs(definition.DirName)
	if pathErr != nil {
		return fmt.Errorf("can't calculate abs path for %v: %v", definition.DirName, pathErr)
	}
	outputPath := filepath.Join(absPath, output)
	writeErr := ioutil.WriteFile(outputPath, src, 0664)
	if writeErr != nil {
		return fmt.Errorf("can't write file to disk: %v", writeErr)
	}
	return nil
}

func (g *Generator) checkObjForInternalPointers(obj types.Object, depth int) (token.Pos, error) {
	pos, objErr := g.checkTypeForInternalPointers(obj.Type(), obj.Pos(), depth)
	if objErr != nil {
		return pos, fmt.Errorf("pointer based obj: '%+v' has %v", obj, objErr)
	}
	return pos, objErr
}

func (g *Generator) checkTypeForInternalPointers(t types.Type, pos token.Pos, depth int) (token.Pos, error) {
	basicType, isBasic := t.Underlying().(*types.Basic)
	if isBasic {
		switch basicType.Kind() {
		case types.String, types.UnsafePointer, types.UntypedString, types.UntypedNil, types.Invalid:
			return pos, fmt.Errorf(
				"pointer based type: '%v'; kind: '%v'",
				t, basicType.Kind(),
			)
		default:
			return token.NoPos, nil
		}
	}
	structType, isStruct := t.Underlying().(*types.Struct)
	if isStruct {
		for i := 0; i < structType.NumFields(); i++ {
			field := structType.Field(i)
			internalPos, fieldErr := g.checkObjForInternalPointers(field, depth+1)
			if fieldErr != nil {
				depthTab := strings.Repeat("\t", depth+1)
				return internalPos, fmt.Errorf(
					"pointer based field '%v' of type '%v':\n%s%v",
					field.Name(), field.Type(), depthTab, fieldErr,
				)
			}
		}
		return token.NoPos, nil
	}
	arrayType, isArray := t.Underlying().(*types.Array)
	if isArray {
		if arrayType.Len() >= 0 {
			errPos, elemErr := g.checkTypeForInternalPointers(arrayType.Elem(), pos, depth+1)
			if elemErr != nil {
				depthTab := strings.Repeat("\t", depth+1)
				return errPos, fmt.Errorf(
					"array of elements %v has pointer based type:\n%s%v",
					t, depthTab, elemErr,
				)
			}
			return token.NoPos, nil
		}
	}
	return pos, fmt.Errorf("pointer based type: '%v'", t)
}
