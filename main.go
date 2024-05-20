package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	_ "github.com/tliron/commonlog/simple"
)

const lsName = "glsl_lsp_go"

var version string = "0.0.1"
var handler protocol.Handler

var documentMutex sync.Mutex
var document string
var functions []FunctionDef
var structs []StructDef
var builtinTypes []string

type FunctionDef struct {
	Name  string
	Range protocol.Range
}

type StructDef struct {
	Name  string
	Range protocol.Range
}

func (self *StructDef) toProtocol() protocol.DocumentSymbol {
	return protocol.DocumentSymbol{
		Name:           self.Name,
		Detail:         new(string),
		Kind:           protocol.SymbolKindClass,
		Tags:           []protocol.SymbolTag{},
		Deprecated:     nil,
		Range:          self.Range,
		SelectionRange: self.Range,
		Children:       []protocol.DocumentSymbol{},
	}
}
func (self *FunctionDef) toProtocol() protocol.DocumentSymbol {
	return protocol.DocumentSymbol{
		Name:           self.Name,
		Detail:         new(string),
		Kind:           protocol.SymbolKindFunction,
		Tags:           []protocol.SymbolTag{},
		Deprecated:     nil,
		Range:          self.Range,
		SelectionRange: self.Range,
		Children:       []protocol.DocumentSymbol{},
	}
}

func main() {
	path := "glsl_lsp.log"
	commonlog.Configure(2, &path)
	commonlog.NewInfoMessage(0, "Startup").Send()

	builtinTypes = make([]string, 16)
	// https://www.khronos.org/opengl/wiki/Data_Type_(GLSL)
	builtinTypes[0] = "bool"
	builtinTypes[1] = "int"
	builtinTypes[2] = "uint"
	builtinTypes[3] = "float"
	for i := range 3 {
		builtinTypes[4+i] = fmt.Sprintf("bvec%d", i+2)
	}
	for i := range 3 {
		builtinTypes[7+i] = fmt.Sprintf("ivec%d", i+2)
	}
	for i := range 3 {
		builtinTypes[10+i] = fmt.Sprintf("uvec%d", i+2)
	}
	for i := range 3 {
		builtinTypes[13+i] = fmt.Sprintf("vec%d", i+2)
	}
	fmt.Println(builtinTypes)

	handler = protocol.Handler{
		Initialize:                 initialize,
		Shutdown:                   shutdown,
		TextDocumentDocumentSymbol: symbolProvider,
		TextDocumentDidOpen:        documentOpen,
		TextDocumentDidSave:        documentSave,
		TextDocumentDidChange:      documentChange,
	}

	server := server.NewServer(&handler, lsName, true)

	server.RunStdio()
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	commonlog.NewInfoMessage(0, "Initializing server...").Send()

	capabilities := handler.CreateServerCapabilities()

	capabilities.DocumentSymbolProvider = protocol.DocumentSymbolOptions{}
	capabilities.TextDocumentSync = protocol.TextDocumentSyncKindFull

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func shutdown(context *glsp.Context) error {
	return nil
}

func symbolProvider(context *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	_ = params
	var symbols []protocol.DocumentSymbol
	// prealloc with make

	for _, s := range structs {
		symbols = append(symbols, s.toProtocol())
	}
	for _, s := range functions {
		symbols = append(symbols, s.toProtocol())
	}

	return symbols, nil
}

func documentOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	documentMutex.Lock()
	defer documentMutex.Unlock()
	document = params.TextDocument.Text

	s, err := extractStructs(document)
	if err != nil {
		return err
	}

	f, err := extractFunctions(document, s)
	if err != nil {
		return err
	}
	structs = s
	functions = f

	return nil
}

func documentChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	d, ok := params.ContentChanges[0].(protocol.TextDocumentContentChangeEventWhole)
	if !ok {
		return errors.New("Failed to parse document change")
	}
	documentMutex.Lock()
	defer documentMutex.Unlock()

	s, err := extractStructs(d.Text)
	if err != nil {
		return err
	}
	structs = s
	f, err := extractFunctions(document, s)
	if err != nil {
		return err
	}
	structs = s
	functions = f

	document = d.Text
	return nil
}

func extractStructs(code string) ([]StructDef, error) {
	pattern := regexp.MustCompile(`struct\s+(\w+)\s*\{`)
	matches := pattern.FindAllStringSubmatchIndex(code, -1)

	defs := make([]StructDef, 0)
	for _, idx := range matches {
		name := string([]rune(code)[idx[2]:idx[3]])
		line, offset := convertToLine(code, idx[2])
		defs = append(defs, StructDef{
			Name: name,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(line),
					Character: uint32(idx[2] - offset - 1),
				},
				End: protocol.Position{
					Line:      uint32(line),
					Character: uint32(idx[3] - offset - 2),
				},
			},
		})

	}
	return defs, nil
}

func extractFunctions(code string, structs []StructDef) ([]FunctionDef, error) {
	returnTypes := make([]string, 0)
	for _, t := range builtinTypes {
		returnTypes = append(returnTypes, t)
	}
	for _, t := range structs {
		returnTypes = append(returnTypes, t.Name)
	}
	returnTypes = append(returnTypes, "void")

	returnTypesPrefix := strings.Join(returnTypes, "|")

	combined := fmt.Sprintf(`(%s)\s+(\w+)\s*\(`, returnTypesPrefix)

	pattern := regexp.MustCompile(combined)
	matches := pattern.FindAllStringSubmatchIndex(code, -1)

	defs := make([]FunctionDef, 0)
	for _, idx := range matches {
		name := string([]rune(code)[idx[4]:idx[5]])
		line, offset := convertToLine(code, idx[4])
		defs = append(defs, FunctionDef{
			Name: name,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(line),
					Character: uint32(idx[4] - offset - 1),
				},
				End: protocol.Position{
					Line:      uint32(line),
					Character: uint32(idx[5] - offset - 2),
				},
			},
		})

	}
	return defs, nil
}

// returns line number and offset of last newline before psoition
func convertToLine(code string, position int) (int, int) {
	until := string([]rune(code)[0:position])
	lineCount := strings.Count(until, "\n")
	offset := strings.LastIndex(until, "\n")
	return lineCount, offset
}

func documentSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	return nil
}
