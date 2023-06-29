{{if .Table.IsJoinTable -}}
{{else -}}
{{- $alias := .Aliases.Table .Table.Name -}}
var (
	{{$alias.DownSingular}}AllColumns               = []string{{"{"}}{{.Table.Columns | columnNames | stringMap .StringFuncs.quoteWrap | join ", "}}{{"}"}}
	{{$alias.DownSingular}}ColumnsWithoutDefault = []string{{"{"}}{{.Table.Columns | filterColumnsByDefault false | columnNames | stringMap .StringFuncs.quoteWrap | join ","}}{{"}"}}
	{{$alias.DownSingular}}ColumnsWithDefault    = []string{{"{"}}{{.Table.Columns | filterColumnsByDefault true | columnNames | stringMap .StringFuncs.quoteWrap | join ","}}{{"}"}}
	{{if .Table.IsView -}}
	{{$alias.DownSingular}}PrimaryKeyColumns     = []string{}
	{{else -}}
	{{$alias.DownSingular}}PrimaryKeyColumns     = []string{{"{"}}{{.Table.PKey.Columns | stringMap .StringFuncs.quoteWrap | join ", "}}{{"}"}}
	{{end -}}
	{{$alias.DownSingular}}GeneratedColumns = []string{{"{"}}{{.Table.Columns | filterColumnsByAuto true | columnNames | stringMap .StringFuncs.quoteWrap | join ","}}{{"}"}}
)

type (
	// {{$alias.UpSingular}}Slice is an alias for a slice of pointers to {{$alias.UpSingular}}.
	// This should almost always be used instead of []{{$alias.UpSingular}}.
	{{$alias.UpSingular}}Slice []*{{$alias.UpSingular}}
	{{if not .NoHooks -}}
	// {{$alias.UpSingular}}Hook is the signature for custom {{$alias.UpSingular}} hook methods
	{{$alias.UpSingular}}Hook func({{if .NoContext}}boil.Executor{{else}}context.Context, boil.ContextExecutor{{end}}, *{{$alias.UpSingular}}) error
	{{- end}}

	{{$alias.DownSingular}}Query struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	{{$alias.DownSingular}}Type = reflect.TypeOf(&{{$alias.UpSingular}}{})
	{{$alias.DownSingular}}Mapping = queries.MakeStructMapping({{$alias.DownSingular}}Type)
	{{if not .Table.IsView -}}
	{{$alias.DownSingular}}PrimaryKeyMapping, _ = queries.BindMapping({{$alias.DownSingular}}Type, {{$alias.DownSingular}}Mapping, {{$alias.DownSingular}}PrimaryKeyColumns)
	{{end -}}
	{{$alias.DownSingular}}InsertCacheMut sync.RWMutex
	{{$alias.DownSingular}}InsertCache = make(map[string]insertCache)
	{{$alias.DownSingular}}UpdateCacheMut sync.RWMutex
	{{$alias.DownSingular}}UpdateCache = make(map[string]updateCache)
	{{$alias.DownSingular}}UpsertCacheMut sync.RWMutex
	{{$alias.DownSingular}}UpsertCache = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
	{{if .Table.IsView -}}
	// These are used in some views
	_ = fmt.Sprintln("")
	_ = reflect.Int
	_ = strings.Builder{}
	_ = sync.Mutex{}
	_ = strmangle.Plural("")
	_ = strconv.IntSize
	{{- end}}
)
{{end -}}
