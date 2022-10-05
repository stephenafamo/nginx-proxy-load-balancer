// Package boilingcore has types and methods useful for generating code that
// acts as a fully dynamic ORM might.
package boilingcore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/sqlboiler/v4/drivers"
	"github.com/volatiletech/sqlboiler/v4/importers"
	"github.com/volatiletech/sqlboiler/v4/templatebin"
	"github.com/volatiletech/strmangle"
)

const (
	templatesDirectory          = "templates"
	templatesSingletonDirectory = "templates/singleton"

	templatesTestDirectory          = "templates_test"
	templatesSingletonTestDirectory = "templates_test/singleton"

	templatesTestMainDirectory = "templates_test/main_test"
)

var (
	// Tags must be in a format like: json, xml, etc.
	rgxValidTag = regexp.MustCompile(`[a-zA-Z_\.]+`)
	// Column names must be in format column_name or table_name.column_name
	rgxValidTableColumn = regexp.MustCompile(`^[\w]+\.[\w]+$|^[\w]+$`)
)

// State holds the global data needed by most pieces to run
type State struct {
	Config *Config

	Driver  drivers.Interface
	Schema  string
	Tables  []drivers.Table
	Dialect drivers.Dialect

	Templates     *templateList
	TestTemplates *templateList
}

// New creates a new state based off of the config
func New(config *Config) (*State, error) {
	s := &State{
		Config: config,
	}

	var templates []lazyTemplate

	defer func() {
		if s.Config.Debug {
			debugOut := struct {
				Config       *Config         `json:"config"`
				DriverConfig drivers.Config  `json:"driver_config"`
				Schema       string          `json:"schema"`
				Dialect      drivers.Dialect `json:"dialect"`
				Tables       []drivers.Table `json:"tables"`
				Templates    []lazyTemplate  `json:"templates"`
			}{
				Config:       s.Config,
				DriverConfig: s.Config.DriverConfig,
				Schema:       s.Schema,
				Dialect:      s.Dialect,
				Tables:       s.Tables,
				Templates:    templates,
			}

			b, err := json.Marshal(debugOut)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", b)
		}
	}()

	if len(config.Version) > 0 {
		noEditDisclaimer = []byte(
			fmt.Sprintf(noEditDisclaimerFmt, " "+config.Version+" "),
		)
	}

	s.Driver = drivers.GetDriver(config.DriverName)

	err := s.initDBInfo(config.DriverConfig)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize tables")
	}

	if err := s.mergeDriverImports(); err != nil {
		return nil, errors.Wrap(err, "unable to merge imports from driver")
	}

	if !s.Config.NoContext {
		s.Config.Imports.All.Standard = append(s.Config.Imports.All.Standard, `"context"`)
		s.Config.Imports.Test.Standard = append(s.Config.Imports.Test.Standard, `"context"`)
	}

	if err := s.processTypeReplacements(); err != nil {
		return nil, err
	}

	templates, err = s.initTemplates()
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize templates")
	}

	err = s.initOutFolders(templates)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize the output folders")
	}

	err = s.initTags(config.Tags)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize struct tags")
	}

	err = s.initAliases(&config.Aliases)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize aliases")
	}

	return s, nil
}

// Run executes the sqlboiler templates and outputs them to files based on the
// state given.
func (s *State) Run() error {
	data := &templateData{
		Tables:            s.Tables,
		Aliases:           s.Config.Aliases,
		DriverName:        s.Config.DriverName,
		PkgName:           s.Config.PkgName,
		AddGlobal:         s.Config.AddGlobal,
		AddPanic:          s.Config.AddPanic,
		AddSoftDeletes:    s.Config.AddSoftDeletes,
		NoContext:         s.Config.NoContext,
		NoHooks:           s.Config.NoHooks,
		NoAutoTimestamps:  s.Config.NoAutoTimestamps,
		NoRowsAffected:    s.Config.NoRowsAffected,
		NoDriverTemplates: s.Config.NoDriverTemplates,
		NoBackReferencing: s.Config.NoBackReferencing,
		StructTagCasing:   s.Config.StructTagCasing,
		TagIgnore:         make(map[string]struct{}),
		Tags:              s.Config.Tags,
		RelationTag:       s.Config.RelationTag,
		Dialect:           s.Dialect,
		Schema:            s.Schema,
		LQ:                strmangle.QuoteCharacter(s.Dialect.LQ),
		RQ:                strmangle.QuoteCharacter(s.Dialect.RQ),
		OutputDirDepth:    s.Config.OutputDirDepth(),

		DBTypes:     make(once),
		StringFuncs: templateStringMappers,
	}

	for _, v := range s.Config.TagIgnore {
		if !rgxValidTableColumn.MatchString(v) {
			return errors.New("Invalid column name %q supplied, only specify column name or table.column, eg: created_at, user.password")
		}
		data.TagIgnore[v] = struct{}{}
	}

	if err := generateSingletonOutput(s, data); err != nil {
		return errors.Wrap(err, "singleton template output")
	}

	if !s.Config.NoTests {
		if err := generateSingletonTestOutput(s, data); err != nil {
			return errors.Wrap(err, "unable to generate singleton test template output")
		}
	}

	var regularDirExtMap, testDirExtMap dirExtMap
	regularDirExtMap = groupTemplates(s.Templates)
	if !s.Config.NoTests {
		testDirExtMap = groupTemplates(s.TestTemplates)
	}

	for _, table := range s.Tables {
		if table.IsJoinTable {
			continue
		}

		data.Table = table

		// Generate the regular templates
		if err := generateOutput(s, regularDirExtMap, data); err != nil {
			return errors.Wrap(err, "unable to generate output")
		}

		// Generate the test templates
		if !s.Config.NoTests {
			if err := generateTestOutput(s, testDirExtMap, data); err != nil {
				return errors.Wrap(err, "unable to generate test output")
			}
		}
	}

	return nil
}

// Cleanup closes any resources that must be closed
func (s *State) Cleanup() error {
	// Nothing here atm, used to close the driver
	return nil
}

// initTemplates loads all template folders into the state object.
//
// If TemplateDirs is set it uses those, else it pulls from assets.
// Then it allows drivers to override, followed by replacements.
//
// Because there's the chance for windows paths to jumped in
// all paths are converted to the native OS's slash style.
//
// Later, in order to properly look up imports the paths will
// be forced back to linux style paths.
func (s *State) initTemplates() ([]lazyTemplate, error) {
	var err error

	templates := make(map[string]templateLoader)
	if len(s.Config.TemplateDirs) != 0 {
		for _, dir := range s.Config.TemplateDirs {
			abs, err := filepath.Abs(dir)
			if err != nil {
				return nil, errors.Wrap(err, "could not find abs dir of templates directory")
			}

			base := filepath.Base(abs)
			root := filepath.Dir(abs)
			tpls, err := findTemplates(root, base)
			if err != nil {
				return nil, err
			}

			mergeTemplates(templates, tpls)
		}
	} else {
		for _, a := range templatebin.AssetNames() {
			if strings.HasSuffix(a, ".tpl") {
				templates[normalizeSlashes(a)] = assetLoader(a)
			}
		}
	}

	if !s.Config.NoDriverTemplates {
		driverTemplates, err := s.Driver.Templates()
		if err != nil {
			return nil, err
		}

		for template, contents := range driverTemplates {
			templates[normalizeSlashes(template)] = base64Loader(contents)
		}
	}

	for _, replace := range s.Config.Replacements {
		splits := strings.Split(replace, ";")
		if len(splits) != 2 {
			return nil, errors.Errorf("replace parameters must have 2 arguments, given: %s", replace)
		}

		original, replacement := normalizeSlashes(splits[0]), splits[1]

		_, ok := templates[original]
		if !ok {
			return nil, errors.Errorf("replace can only replace existing templates, %s does not exist", original)
		}

		templates[original] = fileLoader(replacement)
	}

	// For stability, sort keys to traverse the map and turn it into a slice
	keys := make([]string, 0, len(templates))
	for k := range templates {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lazyTemplates := make([]lazyTemplate, 0, len(templates))
	for _, k := range keys {
		lazyTemplates = append(lazyTemplates, lazyTemplate{
			Name:   k,
			Loader: templates[k],
		})
	}

	s.Templates, err = loadTemplates(lazyTemplates, false)
	if err != nil {
		return nil, err
	}

	if !s.Config.NoTests {
		s.TestTemplates, err = loadTemplates(lazyTemplates, true)
		if err != nil {
			return nil, err
		}
	}

	return lazyTemplates, nil
}

type dirExtMap map[string]map[string][]string

// groupTemplates takes templates and groups them according to their output directory
// and file extension.
func groupTemplates(templates *templateList) dirExtMap {
	tplNames := templates.Templates()
	dirs := make(map[string]map[string][]string)
	for _, tplName := range tplNames {
		normalized, isSingleton, _, _ := outputFilenameParts(tplName)
		if isSingleton {
			continue
		}

		dir := filepath.Dir(normalized)
		if dir == "." {
			dir = ""
		}

		extensions, ok := dirs[dir]
		if !ok {
			extensions = make(map[string][]string)
			dirs[dir] = extensions
		}

		ext := getLongExt(tplName)
		ext = strings.TrimSuffix(ext, ".tpl")
		slice := extensions[ext]
		extensions[ext] = append(slice, tplName)
	}

	return dirs
}

// findTemplates uses a root path: (/home/user/gopath/src/../sqlboiler/)
// and a base path: /templates
// to create a bunch of file loaders of the form:
// templates/00_struct.tpl -> /absolute/path/to/that/file
// Note the missing leading slash, this is important for the --replace argument
func findTemplates(root, base string) (map[string]templateLoader, error) {
	templates := make(map[string]templateLoader)
	rootBase := filepath.Join(root, base)
	err := filepath.Walk(rootBase, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".tpl" {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return errors.Wrapf(err, "could not find relative path to base root: %s", rootBase)
		}

		relative = strings.TrimLeft(relative, string(os.PathSeparator))
		templates[relative] = fileLoader(path)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return templates, nil
}

// initDBInfo retrieves information about the database
func (s *State) initDBInfo(config map[string]interface{}) error {
	dbInfo, err := s.Driver.Assemble(config)
	if err != nil {
		return errors.Wrap(err, "unable to fetch table data")
	}

	if len(dbInfo.Tables) == 0 {
		return errors.New("no tables found in database")
	}

	if err := checkPKeys(dbInfo.Tables); err != nil {
		return err
	}

	s.Schema = dbInfo.Schema
	s.Tables = dbInfo.Tables
	s.Dialect = dbInfo.Dialect

	return nil
}

// mergeDriverImports calls the driver and asks for its set
// of imports, then merges it into the current configuration's
// imports.
func (s *State) mergeDriverImports() error {
	drivers, err := s.Driver.Imports()
	if err != nil {
		return errors.Wrap(err, "failed to fetch driver's imports")
	}

	s.Config.Imports = importers.Merge(s.Config.Imports, drivers)
	return nil
}

// processTypeReplacements checks the config for type replacements
// and performs them.
func (s *State) processTypeReplacements() error {
	for _, r := range s.Config.TypeReplaces {

		for i := range s.Tables {
			t := s.Tables[i]

			if !shouldReplaceInTable(t, r) {
				continue
			}

			for j := range t.Columns {
				c := t.Columns[j]
				if matchColumn(c, r.Match) {
					t.Columns[j] = columnMerge(c, r.Replace)

					if len(r.Imports.Standard) != 0 || len(r.Imports.ThirdParty) != 0 {
						s.Config.Imports.BasedOnType[t.Columns[j].Type] = importers.Set{
							Standard:   r.Imports.Standard,
							ThirdParty: r.Imports.ThirdParty,
						}
					}
				}
			}
		}
	}

	return nil
}

// matchColumn checks if a column 'c' matches specifiers in 'm'.
// Anything defined in m is checked against a's values, the
// match is a done using logical and (all specifiers must match).
// Bool fields are only checked if a string type field matched first
// and if a string field matched they are always checked (must be defined).
//
// Doesn't care about Unique columns since those can vary independent of type.
func matchColumn(c, m drivers.Column) bool {
	matchedSomething := false

	// return true if we matched, or we don't have to match
	// if we actually matched against something, then additionally set
	// matchedSomething so we can check boolean values too.
	matches := func(matcher, value string) bool {
		if len(matcher) != 0 && matcher != value {
			return false
		}
		matchedSomething = true
		return true
	}

	if !matches(m.Name, c.Name) {
		return false
	}
	if !matches(m.Type, c.Type) {
		return false
	}
	if !matches(m.DBType, c.DBType) {
		return false
	}
	if !matches(m.UDTName, c.UDTName) {
		return false
	}
	if !matches(m.FullDBType, c.FullDBType) {
		return false
	}
	if m.ArrType != nil && (c.ArrType == nil || !matches(*m.ArrType, *c.ArrType)) {
		return false
	}
	if m.DomainName != nil && (c.DomainName == nil || !matches(*m.DomainName, *c.DomainName)) {
		return false
	}

	if !matchedSomething {
		return false
	}

	if m.AutoGenerated != c.AutoGenerated {
		return false
	}
	if m.Nullable != c.Nullable {
		return false
	}

	return true
}

// columnMerge merges values from src into dst. Bools are copied regardless
// strings are copied if they have values. Name is excluded because it doesn't make
// sense to non-programatically replace a name.
func columnMerge(dst, src drivers.Column) drivers.Column {
	ret := dst
	if len(src.Type) != 0 {
		ret.Type = src.Type
	}
	if len(src.DBType) != 0 {
		ret.DBType = src.DBType
	}
	if len(src.UDTName) != 0 {
		ret.UDTName = src.UDTName
	}
	if len(src.FullDBType) != 0 {
		ret.FullDBType = src.FullDBType
	}
	if src.ArrType != nil && len(*src.ArrType) != 0 {
		ret.ArrType = new(string)
		*ret.ArrType = *src.ArrType
	}

	return ret
}

// shouldReplaceInTable checks if tables were specified in types.match in the config.
// If tables were set, it checks if the given table is among the specified tables.
func shouldReplaceInTable(t drivers.Table, r TypeReplace) bool {
	if len(r.Tables) == 0 {
		return true
	}

	for _, replaceInTable := range r.Tables {
		if replaceInTable == t.Name {
			return true
		}
	}

	return false
}

// initOutFolders creates the folders that will hold the generated output.
func (s *State) initOutFolders(lazyTemplates []lazyTemplate) error {
	if s.Config.Wipe {
		if err := os.RemoveAll(s.Config.OutFolder); err != nil {
			return err
		}
	}

	newDirs := make(map[string]struct{})
	for _, t := range lazyTemplates {
		// templates/js/00_struct.js.tpl
		// templates/js/singleton/00_struct.js.tpl
		// we want the js part only
		fragments := strings.Split(t.Name, string(os.PathSeparator))

		// Throw away the root dir and filename
		fragments = fragments[1 : len(fragments)-1]
		if len(fragments) != 0 && fragments[len(fragments)-1] == "singleton" {
			fragments = fragments[:len(fragments)-1]
		}

		if len(fragments) == 0 {
			continue
		}

		newDirs[strings.Join(fragments, string(os.PathSeparator))] = struct{}{}
	}

	if err := os.MkdirAll(s.Config.OutFolder, os.ModePerm); err != nil {
		return err
	}

	for d := range newDirs {
		if err := os.MkdirAll(filepath.Join(s.Config.OutFolder, d), os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

// initTags removes duplicate tags and validates the format
// of all user tags are simple strings without quotes: [a-zA-Z_\.]+
func (s *State) initTags(tags []string) error {
	s.Config.Tags = strmangle.RemoveDuplicates(s.Config.Tags)
	for _, v := range s.Config.Tags {
		if !rgxValidTag.MatchString(v) {
			return errors.New("Invalid tag format %q supplied, only specify name, eg: xml")
		}
	}

	return nil
}

func (s *State) initAliases(a *Aliases) error {
	FillAliases(a, s.Tables)
	return nil
}

// checkPKeys ensures every table has a primary key column
func checkPKeys(tables []drivers.Table) error {
	var missingPkey []string
	for _, t := range tables {
		if t.PKey == nil {
			missingPkey = append(missingPkey, t.Name)
		}
	}

	if len(missingPkey) != 0 {
		return errors.Errorf("primary key missing in tables (%s)", strings.Join(missingPkey, ", "))
	}

	return nil
}

func mergeTemplates(dst, src map[string]templateLoader) {
	for k, v := range src {
		dst[k] = v
	}
}

// normalizeSlashes takes a path that was made on linux or windows and converts it
// to a native path.
func normalizeSlashes(path string) string {
	path = strings.ReplaceAll(path, `/`, string(os.PathSeparator))
	path = strings.ReplaceAll(path, `\`, string(os.PathSeparator))
	return path
}

// denormalizeSlashes takes any backslashes and converts them to linux style slashes
func denormalizeSlashes(path string) string {
	path = strings.ReplaceAll(path, `\`, `/`)
	return path
}
