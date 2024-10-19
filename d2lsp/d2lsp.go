// d2lsp contains functions useful for IDE clients
package d2lsp

import (
	"fmt"
	"strings"

	"oss.terrastruct.com/d2/d2ast"
	"oss.terrastruct.com/d2/d2ir"
	"oss.terrastruct.com/d2/d2parser"
	"oss.terrastruct.com/d2/lib/memfs"
)

func GetRefRanges(path string, fs map[string]string, boardPath []string, key string) (ranges []d2ast.Range, importRanges []d2ast.Range, _ error) {
	m, err := getBoardMap(path, fs, boardPath)
	if err != nil {
		return nil, nil, err
	}

	mk, err := d2parser.ParseMapKey(key)
	if err != nil {
		return nil, nil, err
	}
	if mk.Key == nil && len(mk.Edges) == 0 {
		return nil, nil, fmt.Errorf(`"%s" is invalid`, key)
	}

	var f *d2ir.Field
	if mk.Key != nil {
		for _, p := range mk.Key.Path {
			f = m.GetField(p.Unbox().ScalarString())
			if f == nil {
				return nil, nil, nil
			}
			m = f.Map()
		}
	}

	if len(mk.Edges) > 0 {
		eids := d2ir.NewEdgeIDs(mk)
		var edges []*d2ir.Edge
		for _, eid := range eids {
			edges = append(edges, m.GetEdges(eid, nil, nil)...)
		}
		if len(edges) == 0 {
			return nil, nil, nil
		}
		for _, edge := range edges {
			for _, ref := range edge.References {
				ranges = append(ranges, ref.AST().GetRange())
			}
			if edge.ImportAST() != nil {
				importRanges = append(importRanges, edge.ImportAST().GetRange())
			}
		}
	} else {
		for _, ref := range f.References {
			ranges = append(ranges, ref.AST().GetRange())
		}
		if f.ImportAST() != nil {
			importRanges = append(importRanges, f.ImportAST().GetRange())
		}
	}
	return ranges, importRanges, nil
}

func getBoardMap(path string, fs map[string]string, boardPath []string) (*d2ir.Map, error) {
	if _, ok := fs[path]; !ok {
		return nil, fmt.Errorf(`"%s" not found`, path)
	}
	r := strings.NewReader(fs[path])
	ast, err := d2parser.Parse(path, r, nil)
	if err != nil {
		return nil, err
	}

	mfs, err := memfs.New(fs)
	if err != nil {
		return nil, err
	}

	m, _, err := d2ir.Compile(ast, &d2ir.CompileOptions{
		FS: mfs,
	})
	if err != nil {
		return nil, err
	}

	m = m.FindBoardRoot(boardPath)
	if m == nil {
		return nil, fmt.Errorf(`board "%v" not found`, boardPath)
	}
	return m, nil
}
