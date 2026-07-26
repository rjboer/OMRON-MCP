package mcp

import "github.com/rjboer/omron-mcp/internal/sysmac"

func sysmacTransactionTools() []tool {
	return []tool{
		{Name: "sysmac_transaction_start", Description: "Start a multi-file project transaction in a staging copy.", InputSchema: objectSchema(map[string]any{"project_path": stringProperty("Sysmac Solution directory")}, "project_path")},
		{Name: "sysmac_transaction_stage_file", Description: "Stage replacement bytes for a relative project file without touching the source.", InputSchema: objectSchema(map[string]any{"transaction_id": stringProperty("Transaction ID"), "relative_path": stringProperty("Path relative to project root"), "content": stringProperty("Replacement file content")}, "transaction_id", "relative_path", "content")},
		{Name: "sysmac_transaction_delete_file", Description: "Stage deletion of a relative project file without touching the source.", InputSchema: objectSchema(map[string]any{"transaction_id": stringProperty("Transaction ID"), "relative_path": stringProperty("Path relative to project root")}, "transaction_id", "relative_path")},
		{Name: "sysmac_transaction_preview", Description: "Preview staged project changes without committing them.", InputSchema: objectSchema(map[string]any{"transaction_id": stringProperty("Transaction ID")}, "transaction_id")},
		{Name: "sysmac_transaction_commit", Description: "Commit a staged multi-file transaction after hash validation. Requires explicit confirmation.", InputSchema: objectSchema(map[string]any{"transaction_id": stringProperty("Transaction ID"), "confirm_commit": map[string]any{"type": "boolean", "description": "Must be true to commit staged changes"}}, "transaction_id", "confirm_commit")},
		{Name: "sysmac_transaction_rollback", Description: "Discard a staged project transaction.", InputSchema: objectSchema(map[string]any{"transaction_id": stringProperty("Transaction ID")}, "transaction_id")},
	}
}

func sysmacTransactionToolHandlers() map[string]sysmacToolHandler {
	return map[string]sysmacToolHandler{
		"sysmac_transaction_start":       handleTransactionStart,
		"sysmac_transaction_stage_file":  handleTransactionStageFile,
		"sysmac_transaction_delete_file": handleTransactionDeleteFile,
		"sysmac_transaction_preview":     handleTransactionPreview,
		"sysmac_transaction_commit":      handleTransactionCommit,
		"sysmac_transaction_rollback":    handleTransactionRollback,
	}
}

func handleTransactionStart(call sysmacToolCall) (map[string]any, error) {
	path, err := call.string("project_path", true)
	if err != nil {
		return nil, err
	}
	tx, err := sysmac.StartProjectTransaction(path)
	if err != nil {
		return nil, err
	}
	id := storeTransaction(tx)
	return toolSuccess(map[string]any{"status": "started", "transaction_id": id, "project_path": tx.SourceRoot, "staging_path": tx.StageRoot}), nil
}

func handleTransactionPreview(call sysmacToolCall) (map[string]any, error) {
	id, err := call.string("transaction_id", true)
	if err != nil {
		return nil, err
	}
	tx, err := getTransaction(id)
	if err != nil {
		return nil, err
	}
	changes, err := tx.Preview()
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "preview", "transaction_id": id, "changes": changes}), nil
}

func handleTransactionStageFile(call sysmacToolCall) (map[string]any, error) {
	id, err := call.string("transaction_id", true)
	if err != nil {
		return nil, err
	}
	relative, err := call.string("relative_path", true)
	if err != nil {
		return nil, err
	}
	content, err := call.string("content", true)
	if err != nil {
		return nil, err
	}
	tx, err := getTransaction(id)
	if err != nil {
		return nil, err
	}
	if err := tx.StageFile(relative, []byte(content)); err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "staged", "transaction_id": id, "relative_path": relative}), nil
}

func handleTransactionDeleteFile(call sysmacToolCall) (map[string]any, error) {
	id, err := call.string("transaction_id", true)
	if err != nil {
		return nil, err
	}
	relative, err := call.string("relative_path", true)
	if err != nil {
		return nil, err
	}
	tx, err := getTransaction(id)
	if err != nil {
		return nil, err
	}
	if err := tx.DeleteFile(relative); err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "deletion_staged", "transaction_id": id, "relative_path": relative}), nil
}

func handleTransactionCommit(call sysmacToolCall) (map[string]any, error) {
	id, err := call.string("transaction_id", true)
	if err != nil {
		return nil, err
	}
	if err := call.confirmed("confirm_commit", "confirm_commit must be true to commit a transaction"); err != nil {
		return nil, err
	}
	tx, err := getTransaction(id)
	if err != nil {
		return nil, err
	}
	changes, err := tx.Preview()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	removeTransaction(id)
	return toolSuccess(map[string]any{"status": "committed", "transaction_id": id, "changes": changes}), nil
}

func handleTransactionRollback(call sysmacToolCall) (map[string]any, error) {
	id, err := call.string("transaction_id", true)
	if err != nil {
		return nil, err
	}
	tx, err := getTransaction(id)
	if err != nil {
		return nil, err
	}
	if err := tx.Rollback(); err != nil {
		return nil, err
	}
	removeTransaction(id)
	return toolSuccess(map[string]any{"status": "rolled_back", "transaction_id": id}), nil
}
