{{- if or .Table.IsJoinTable .Table.IsView -}}
{{- else -}}
	{{- range $rel := .Table.ToManyRelationships -}}
		{{- $ltable := $.Aliases.Table $rel.Table -}}
		{{- $ftable := $.Aliases.Table $rel.ForeignTable -}}
		{{- $relAlias := $.Aliases.ManyRelationship $rel.ForeignTable $rel.Name $rel.JoinTable $rel.JoinLocalFKeyName -}}
		{{- $schemaForeignTable := .ForeignTable | $.SchemaTable -}}
		{{- $canSoftDelete := (getTable $.Tables .ForeignTable).CanSoftDelete $.AutoColumns.Deleted}}
// {{$relAlias.Local}} retrieves all the {{.ForeignTable | singular}}'s {{$ftable.UpPlural}} with an executor
{{- if not (eq $relAlias.Local $ftable.UpPlural)}} via {{$rel.ForeignColumn}} column{{- end}}.
func (o *{{$ltable.UpSingular}}) {{$relAlias.Local}}(mods ...qm.QueryMod) {{$ftable.DownSingular}}Query {
	var queryMods []qm.QueryMod
	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

		{{if $rel.ToJoinTable -}}
	queryMods = append(queryMods,
		{{$schemaJoinTable := $rel.JoinTable | $.SchemaTable -}}
		qm.InnerJoin("{{$schemaJoinTable}} on {{$schemaForeignTable}}.{{$rel.ForeignColumn | $.Quotes}} = {{$schemaJoinTable}}.{{$rel.JoinForeignColumn | $.Quotes}}"),
		qm.Where("{{$schemaJoinTable}}.{{$rel.JoinLocalColumn | $.Quotes}}=?", o.{{$ltable.Column $rel.Column}}),
	)
		{{else -}}
	queryMods = append(queryMods,
		qm.Where("{{$schemaForeignTable}}.{{$rel.ForeignColumn | $.Quotes}}=?", o.{{$ltable.Column $rel.Column}}),
	)
		{{end}}

	return {{$ftable.UpPlural}}(queryMods...)
}

{{end -}}{{- /* range relationships */ -}}
{{- end -}}{{- /* if isJoinTable */ -}}
