package nativec

import (
	"bytes"
	"fmt"
	"sort"
)

// EmbeddedAsset is compiled directly into a native executable by Z15 app builds.
type EmbeddedAsset struct {
	Name string
	Data []byte
}

func generateEmbeddedAssetsSource(assets []EmbeddedAsset) []byte {
	assets = append([]EmbeddedAsset(nil), assets...)
	sort.Slice(assets, func(i, j int) bool { return assets[i].Name < assets[j].Name })
	var out bytes.Buffer
	out.WriteString("#include <stddef.h>\n#include <string.h>\n\n")
	for index, asset := range assets {
		fmt.Fprintf(&out, "static const unsigned char z_asset_%d[] = {", index)
		if len(asset.Data) == 0 {
			out.WriteString("0")
		} else {
			for offset, value := range asset.Data {
				if offset%16 == 0 {
					out.WriteString("\n    ")
				}
				fmt.Fprintf(&out, "0x%02x,", value)
			}
		}
		out.WriteString("\n};\n")
	}
	out.WriteString("\ntypedef struct { const char *name; const unsigned char *data; size_t size; } ZEmbeddedAssetEntry;\n")
	fmt.Fprintf(&out, "static const ZEmbeddedAssetEntry z_embedded_assets[%d] = {\n", maxInt(1, len(assets)))
	if len(assets) == 0 {
		out.WriteString("    {NULL, NULL, 0},\n")
	} else {
		for index, asset := range assets {
			fmt.Fprintf(&out, "    {%q, z_asset_%d, %d},\n", asset.Name, index, len(asset.Data))
		}
	}
	out.WriteString("};\n\n")
	fmt.Fprintf(&out, "size_t z_embedded_asset_count(void) { return %d; }\n", len(assets))
	out.WriteString("const char *z_embedded_asset_name(size_t index) { return index < z_embedded_asset_count() ? z_embedded_assets[index].name : NULL; }\n")
	out.WriteString("const unsigned char *z_embedded_asset_data(size_t index, size_t *size) { if(index >= z_embedded_asset_count()) { if(size) *size = 0; return NULL; } if(size) *size = z_embedded_assets[index].size; return z_embedded_assets[index].data; }\n")
	out.WriteString("const unsigned char *z_embedded_asset_find(const char *name, size_t *size) { if(name == NULL) { if(size) *size = 0; return NULL; } for(size_t i=0;i<z_embedded_asset_count();i++) { if(strcmp(name,z_embedded_assets[i].name)==0) { if(size) *size=z_embedded_assets[i].size; return z_embedded_assets[i].data; } } if(size) *size=0; return NULL; }\n")
	return out.Bytes()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
