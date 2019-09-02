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
)

type allocatorDefinition struct {
	DirName        string
	PkgName        string
	TargetTypeName string
}

func RunGeneratorForTypes(dirName string, targetTypes []string) error {
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
		generationErr := generateAllocators(fset, obj, t)
		if generationErr != nil {
			return fmt.Errorf("can't generate allocator for type: %v: \n%v", obj.Type(), generationErr)
		}
	}
	return nil
}

func generateAllocators(fset *token.FileSet, obj types.Object, typeName string) error {
	checkPos, checkErr := checkObjForInternalPointers(obj, 0)
	if checkErr != nil {
		return fmt.Errorf(
			"target obj '%v' has internal pointers: %v\npointer position: %v\n%v",
			obj.Type(), fset.Position(obj.Pos()), fset.Position(checkPos), checkErr,
		)
	}
	definition := allocatorDefinition{
		DirName:        obj.Pkg().Path(),
		PkgName:        obj.Pkg().Name(),
		TargetTypeName: typeName,
	}
	return generateFromTemplateAndWriteToFile(definition)
}

func generateFromTemplateAndWriteToFile(definition allocatorDefinition) error {
	var b bytes.Buffer
	templateErr := embeddedTemplate.Execute(&b, definition)
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

func checkObjForInternalPointers(obj types.Object, depth int) (token.Pos, error) {
	pos, objErr := checkTypeForInternalPointers(obj.Type(), obj.Pos(), depth)
	if objErr != nil {
		return pos, fmt.Errorf("pointer based obj: '%+v' has %v", obj, objErr)
	}
	return pos, objErr
}

func checkTypeForInternalPointers(t types.Type, pos token.Pos, depth int) (token.Pos, error) {
	basicType, isBasic := t.Underlying().(*types.Basic)
	if isBasic {
		switch basicType.Kind() {
		case types.String, types.UnsafePointer, types.UntypedString, types.UntypedNil, types.Invalid:
			return pos, fmt.Errorf("pointer based type: '%v'; kind: '%v'", t, basicType.Kind())
		default:
			return token.NoPos, nil
		}
	}
	structType, isStruct := t.Underlying().(*types.Struct)
	if isStruct {
		for i := 0; i < structType.NumFields(); i++ {
			field := structType.Field(i)
			pos, fieldErr := checkObjForInternalPointers(field, depth+1)
			if fieldErr != nil {
				depthTab := strings.Repeat("\t", depth+1)
				return pos, fmt.Errorf("pointer based field '%v' of type '%v':\n%s%v", field.Name(), field.Type(), depthTab, fieldErr)
			}
		}
		return token.NoPos, nil
	}
	arrayType, isArray := t.Underlying().(*types.Array)
	if isArray {
		if arrayType.Len() >= 0 {
			errPos, elemErr := checkTypeForInternalPointers(arrayType.Elem(), pos, depth+1)
			if elemErr != nil {
				depthTab := strings.Repeat("\t", depth+1)
				return errPos, fmt.Errorf("array of elements %v has pointer based type:\n%s%v", t, depthTab, elemErr)
			}
			return token.NoPos, nil
		}
	}
	return pos, fmt.Errorf("pointer based type: '%v'", t)
}
