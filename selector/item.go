package selector

import (
	"fmt"
	"io/fs"
	"path"
)

const (
	ITEM_DATA_ID_KEY            = "id"
	ITEM_DATA_COMPRESSED_KEY    = "is_compressed"
	ITEM_DATA_BINARY_TYPE_KEY   = "binary_type"
	ITEM_DATA_DOWNLOAD_PATH_KEY = "download_path"
	ITEM_DATA_FS_PATH_KEY       = "fs_path"
	ITEM_DATA_FS_KEY            = "fs"

	BINARY_TYPE_DEB        = "deb_installer"
	BINARY_TYPE_RPM        = "rpm_installer"
	BINARY_TYPE_EXECUTABLE = "binary"
)

type ItemType int

const (
	ItemId ItemType = iota
	ItemCompressed
	ItemBinaryType
	ItemDownloadPath
	ItemFsPath
	ItemFs
)

func (ss ItemType) String() string {
	switch ss {
	case ItemId:
		return ITEM_DATA_ID_KEY
	case ItemCompressed:
		return ITEM_DATA_COMPRESSED_KEY
	case ItemBinaryType:
		return ITEM_DATA_BINARY_TYPE_KEY
	case ItemDownloadPath:
		return ITEM_DATA_DOWNLOAD_PATH_KEY
	case ItemFsPath:
		return ITEM_DATA_FS_PATH_KEY
	case ItemFs:
		return ITEM_DATA_FS_KEY
	default:
		return fmt.Sprintf("Unknown(%d)", ss)
	}
}

type BinaryType int

const (
	BinaryDebInstaller BinaryType = iota
	BinaryRpmInstaller
	BinaryExecutable
)

func (ss BinaryType) String() string {
	switch ss {
	case BinaryDebInstaller:
		return BINARY_TYPE_DEB
	case BinaryRpmInstaller:
		return BINARY_TYPE_RPM
	case BinaryExecutable:
		return BINARY_TYPE_EXECUTABLE
	default:
		return fmt.Sprintf("Unknown(%d)", ss)
	}
}

func BinaryTypeFromPath(fromPath string) BinaryType {
	extension := path.Ext(fromPath)
	switch extension {
	case ".rpm":
		return BinaryRpmInstaller
	case ".deb":
		return BinaryDebInstaller
	default:
		return BinaryExecutable
	}
}

// selection item with collection of IProperty properties
type SelectorItem struct {
	Name     string
	Selected bool
	Data     map[string]any
}

func (i *SelectorItem) GetData(key string) any {
	return i.Data[key]
}

func (i *SelectorItem) SetData(key string, value any) *SelectorItem {
	i.Data[key] = value
	return i
}

func (i *SelectorItem) GetId() int {
	return i.GetData(ItemId.String()).(int)
}
func (i *SelectorItem) SetId(value int) *SelectorItem {
	i.Data[ItemId.String()] = value
	return i
}

func (i *SelectorItem) GetCompressed() bool {
	return i.GetData(ItemCompressed.String()).(bool)
}
func (i *SelectorItem) SetCompressed(value bool) *SelectorItem {
	i.Data[ItemCompressed.String()] = value
	return i
}

func (i *SelectorItem) GetBinaryType() BinaryType {
	return i.GetData(ItemBinaryType.String()).(BinaryType)
}
func (i *SelectorItem) SetBinaryType(value BinaryType) *SelectorItem {
	i.Data[ItemBinaryType.String()] = value
	return i
}

func (i *SelectorItem) GetDownloadPath() string {
	return i.GetData(ItemDownloadPath.String()).(string)
}
func (i *SelectorItem) SetDownloadPath(value string) *SelectorItem {
	i.Data[ItemDownloadPath.String()] = value
	return i
}

func (i *SelectorItem) GetFsPath() string {
	return i.GetData(ItemFsPath.String()).(string)
}
func (i *SelectorItem) SetFsPath(value string) *SelectorItem {
	i.Data[ItemFsPath.String()] = value
	return i
}

func (i *SelectorItem) GetFs() fs.FS {
	return i.GetData(ItemFs.String()).(fs.FS)
}
func (i *SelectorItem) SetFs(value fs.FS) *SelectorItem {
	i.Data[ItemFs.String()] = value
	return i
}

func MakeSelectorItem(name string, selected bool) *SelectorItem {
	propMap := make(map[string]any)

	return &SelectorItem{
		Name:     name,
		Selected: selected,
		Data:     propMap,
	}
}
