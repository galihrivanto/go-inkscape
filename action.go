package inkscape

import "fmt"

/*
action-list         :  Print a list of actions and exit.
convert-dpi-method  :  Import DPI convert method.
export-area         :  Export area.
export-area-drawing :  Export drawing area.
export-area-page    :  Export page area.
export-area-snap    :  Export snap area to integer values.
export-background   :  Export background color.
export-background-opacity:  Export background opacity.
export-do           :  Do export.
export-dpi          :  Export DPI.
export-filename     :  Export file name.
export-height       :  Export height.
export-id           :  Export id(s).
export-id-only      :  Export id(s) only.
export-ignore-filters:  Export ignore filters.
export-latex        :  Export LaTeX.
export-margin       :  Export margin.
export-overwrite    :  Export over-write file.
export-pdf-version  :  Export PDF version.
export-plain-svg    :  Export as plain SVG.
export-ps-level     :  Export PostScript level.
export-text-to-path :  Export convert text to paths.
export-type         :  Export file type.
export-use-hints    :  Export using saved hints.
export-width        :  Export width.
file-close          :  Close active document.
file-new            :  Open new document using template.
file-open           :  Open file.
inkscape-version    :  Print Inkscape version and exit.
no-convert-baseline :  Import convert text baselines.
object-set-attribute:  Set or update an attribute on selected objects. Usage: object-set-attribute:attribute name, attribute value;
object-set-property :  Set or update a property on selected objects. Usage: object-set-property:property name, property value;
object-to-path      :  Convert shapes to paths.
object-unlink-clones:  Unlink clones and symbols.
open-page           :  Import page number.
query-all           :  Query 'x', 'y', 'width', and 'height'.
query-height        :  Query 'height' value(s) of object(s).
query-width         :  Query 'width' value(s) of object(s).
query-x             :  Query 'x' value(s) of selected objects.
query-y             :  Query 'y' value(s) of selected objects.
quit-inkscape       :  Immediately quit Inkscape.
select              :  Select by ID (Deprecated)
select-all          :  Select all. Options: 'all' (every object including groups), 'layers', 'no-layers' (top level objects in layers), 'groups' (all groups including layers), 'no-groups' (all objects other than groups and layers, default).
select-by-class     :  Select by class
select-by-element   :  Select by SVG element (e.g. 'rect').
select-by-id        :  Select by ID
select-by-selector  :  Select by CSS selector
select-clear        :  Selection clear
select-invert       :  Invert selection. Options: 'all', 'layers', 'no-layers', 'groups', 'no-groups' (default).
select-list         :  Print a list of objects in current selection.
system-data-directory:  Print system data directory and exit.
transform-remove    :  Remove any transforms from selected objects.
transform-rotate    :  Rotate selected objects by degrees.
transform-scale     :  Scale selected objects by scale factor.
transform-translate :  Translate selected objects (dx,dy).
unselect            :  Unselect by ID (Deprecated)
unselect-by-id      :  Unselect by ID
user-data-directory :  Print user data directory and exit.
vacuum-defs         :  Remove unused definitions (gradients, etc.).
verb                :  Execute verb(s).
verb-list           :  Print a list of verbs and exit.
window-close        :  Close the active window.
window-open         :  Open a window for the active document. GUI only.
*/

// DpiMethod define dpi method when converting
type DpiMethod string

// constant dpi method values
const (
	DpiMethodNone          DpiMethod = "none"
	DpiMethodScaleViewbox            = "scale-viewbox"
	DpiMethodScaleDocument           = "scale-document"
)

// ConvertDpiMethod .
func ConvertDpiMethod(method DpiMethod) string {
	return "convert-dpi-method:" + string(method)
}

// ExportArea .
func ExportArea(x0, y0, x1, y1 int) string {
	return fmt.Sprintf("export-area:%d:%d:%d:%d", x0, y0, x1, y1)
}

// ExportFileName .
func ExportFileName(filePath string) string {
	return "export-filename:" + filePath
}

// ExportPdfVersion .
func ExportPdfVersion(version string) string {
	return "export-pdf-version:" + version
}

// ExportDo .
func ExportDo() string {
	return "export-do"
}

// FileOpen .
func FileOpen(filePath string) string {
	return "file-open:" + filePath
}

// FileClose .
func FileClose() string {
	return "file-close"
}

// SelectAll .
func SelectAll() string {
	return "select-all"
}

// SelectByClass .
func SelectByClass(className string) string {
	return "select-by-class:" + className
}

// SelectByElement .
func SelectByElement(elementName string) string {
	return "select-by-element:" + elementName
}

// SelectByID .
func SelectByID(id string) string {
	return "select-by-id:" + id
}

// SelectByCSS .
func SelectByCSS(query string) string {
	return "select-by-selector:" + query
}

// SelectClear .
func SelectClear() string {
	return "select-clear"
}

// InvertOption define option when invert selection
type InvertOption string

// invert selection option
const (
	InvertOptionAll      InvertOption = "all"
	InvertOptionLayers                = "layers"
	InvertOptionNoLayers              = "no-layers"
	InvertOptionGroup                 = "group"
	InvertOptionNoGroup               = "no-group"
)

// SelectInvert .
func SelectInvert(option InvertOption) string {
	return "select-invert:" + string(option)
}

// SelectList .
func SelectList() string {
	return "select-list"
}

// Version print inksscape version and return
func Version() string {
	return "inkscape-version"
}
