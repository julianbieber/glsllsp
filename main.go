package main

import (
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

func main() {
	path := "glsl_lsp.log"
	commonlog.Configure(2, &path)
	commonlog.NewInfoMessage(0, "Startup").Send()

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
	return symbols, nil
}

func documentOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	documentMutex.Lock()
	defer documentMutex.Unlock()
	document = params.TextDocument.Text
	return nil
}

func documentChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	d, ok := params.ContentChanges[0].(string)
	if !ok {
		return nil
	}
	documentMutex.Lock()
	defer documentMutex.Unlock()

	document = d
	return nil
}

func documentSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	return nil
}
