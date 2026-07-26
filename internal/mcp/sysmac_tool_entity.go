package mcp

import "github.com/rjboer/omron-mcp/internal/sysmac"

func sysmacEntityTools() []tool {
	return []tool{
		{Name: "sysmac_entity_mutate", Description: "Create, rename, move, clone, or delete an OEM entity. Requires explicit confirmation.", InputSchema: objectSchema(map[string]any{"oem_path": stringProperty("Native .oem index path"), "operation": stringProperty("create, rename, move, clone, or delete"), "entity_id": stringProperty("Entity ID"), "parent_id": stringProperty("Parent entity ID for create, move, or clone"), "name": stringProperty("Entity name"), "type": stringProperty("Entity type for create"), "subtype": stringProperty("Entity subtype for create"), "confirm_direct_write": map[string]any{"type": "boolean", "description": "Must be true to mutate project files"}, "protected_root": stringProperty("Optional root that must not be written")}, "oem_path", "operation", "entity_id", "confirm_direct_write")},
		{Name: "sysmac_entity_payload_read", Description: "Read the native XML payload for any indexed entity.", InputSchema: objectSchema(map[string]any{"project_path": stringProperty("Sysmac Solution directory"), "entity_id": stringProperty("Entity ID")}, "project_path", "entity_id")},
		{Name: "sysmac_entity_payload_write", Description: "Write the native XML payload for any indexed entity. Requires explicit confirmation.", InputSchema: objectSchema(map[string]any{"project_path": stringProperty("Sysmac Solution directory"), "entity_id": stringProperty("Entity ID"), "payload_xml": stringProperty("Replacement XML payload"), "confirm_direct_write": map[string]any{"type": "boolean", "description": "Must be true to mutate project files"}, "protected_root": stringProperty("Optional root that must not be written")}, "project_path", "entity_id", "payload_xml", "confirm_direct_write")},
	}
}

func sysmacEntityToolHandlers() map[string]sysmacToolHandler {
	return map[string]sysmacToolHandler{
		"sysmac_entity_mutate":        handleEntityMutate,
		"sysmac_entity_payload_read":  handleEntityPayloadRead,
		"sysmac_entity_payload_write": handleEntityPayloadWrite,
	}
}

func handleEntityMutate(call sysmacToolCall) (map[string]any, error) {
	oemPath, err := call.string("oem_path", true)
	if err != nil {
		return nil, err
	}
	operation, err := call.string("operation", true)
	if err != nil {
		return nil, err
	}
	entityID, err := call.string("entity_id", true)
	if err != nil {
		return nil, err
	}
	parentID, err := call.string("parent_id", false)
	if err != nil {
		return nil, err
	}
	name, err := call.string("name", false)
	if err != nil {
		return nil, err
	}
	typeName, err := call.string("type", false)
	if err != nil {
		return nil, err
	}
	subtype, err := call.string("subtype", false)
	if err != nil {
		return nil, err
	}
	if err := call.confirmed("confirm_direct_write", "confirm_direct_write must be true for direct project mutation"); err != nil {
		return nil, err
	}
	protectedRoot, err := call.string("protected_root", false)
	if err != nil {
		return nil, err
	}
	before, err := sysmac.SHA256File(oemPath)
	if err != nil {
		return nil, err
	}
	createdID, err := sysmac.ApplyOEMMutation(oemPath, sysmac.OEMMutation{Operation: operation, EntityID: entityID, ParentID: parentID, Name: name, Type: typeName, Subtype: subtype}, protectedRoot)
	if err != nil {
		return nil, err
	}
	after, err := sysmac.SHA256File(oemPath)
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "applied", "operation": operation, "entity_id": entityID, "result_entity_id": createdID, "changed_files": []string{oemPath, oemPath + ".backup"}, "source_hash_before": before, "source_hash_after": after, "backup_path": oemPath + ".backup"}), nil
}

func handleEntityPayloadRead(call sysmacToolCall) (map[string]any, error) {
	path, err := call.string("project_path", true)
	if err != nil {
		return nil, err
	}
	entityID, err := call.string("entity_id", true)
	if err != nil {
		return nil, err
	}
	data, payloadPath, err := sysmac.ReadEntityPayload(path, entityID)
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "read", "project_path": path, "entity_id": entityID, "payload_path": payloadPath, "payload_xml": string(data)}), nil
}

func handleEntityPayloadWrite(call sysmacToolCall) (map[string]any, error) {
	path, err := call.string("project_path", true)
	if err != nil {
		return nil, err
	}
	entityID, err := call.string("entity_id", true)
	if err != nil {
		return nil, err
	}
	payloadXML, err := call.string("payload_xml", true)
	if err != nil {
		return nil, err
	}
	if err := call.confirmed("confirm_direct_write", "confirm_direct_write must be true for direct project mutation"); err != nil {
		return nil, err
	}
	protectedRoot, err := call.string("protected_root", false)
	if err != nil {
		return nil, err
	}
	payloadPath, err := sysmac.EntityPayloadPath(path, entityID)
	if err != nil {
		return nil, err
	}
	before, err := sysmac.SHA256File(payloadPath)
	if err != nil {
		return nil, err
	}
	backup, err := sysmac.WriteEntityPayloadAtomic(path, entityID, []byte(payloadXML), protectedRoot)
	if err != nil {
		return nil, err
	}
	after, err := sysmac.SHA256File(payloadPath)
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "applied", "project_path": path, "entity_id": entityID, "payload_path": payloadPath, "changed_files": []string{payloadPath, backup}, "source_hash_before": before, "source_hash_after": after, "backup_path": backup}), nil
}
