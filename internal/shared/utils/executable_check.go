package utils

import (
	"path/filepath"
	"strings"
)

// executableExtensions is the denylist of file extensions treated as executable/script uploads.
var executableExtensions = map[string]struct{}{
	".exe": {}, ".msi": {}, ".dmg": {}, ".app": {}, ".deb": {}, ".rpm": {},
	".sh": {}, ".bash": {}, ".zsh": {}, ".fish": {},
	".bat": {}, ".cmd": {}, ".com": {}, ".ps1": {}, ".vbs": {}, ".jse": {},
	".scr": {}, ".pif": {}, ".jar": {}, ".war": {}, ".ear": {},
	".dll": {}, ".so": {}, ".dylib": {},
	".elf": {},
}

// executableMagics is the list of magic byte prefixes for known executable file formats.
// Each entry matches at offset 0 in the file.
var executableMagics = [][]byte{
	{0x4D, 0x5A},             // PE/MZ — Windows EXE, DLL, COM
	{0x7F, 0x45, 0x4C, 0x46}, // ELF — Linux/Unix binary
	{0xCA, 0xFE, 0xBA, 0xBE}, // Mach-O fat binary (macOS)
	{0xCE, 0xFA, 0xED, 0xFE}, // Mach-O 32-bit LE
	{0xCF, 0xFA, 0xED, 0xFE}, // Mach-O 64-bit LE
	{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O 32-bit BE
	{0xFE, 0xED, 0xFA, 0xCF}, // Mach-O 64-bit BE
	{0x23, 0x21},             // Shebang (#!) — shell/script
}

// IsExecutableUploadRejected returns true when the file must be blocked because its extension
// or magic bytes match a known executable or script format.
//
// Parameters:
//   - filename: original name from multipart header; used for extension lookup.
//   - head: first N bytes of the file content (at least 4 bytes recommended); may be nil or short.
func IsExecutableUploadRejected(filename string, head []byte) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, denied := executableExtensions[ext]; denied {
		return true
	}
	for _, magic := range executableMagics {
		if hasMagicPrefix(head, magic) {
			return true
		}
	}
	return false
}

func hasMagicPrefix(buf, magic []byte) bool {
	if len(buf) < len(magic) {
		return false
	}
	for i, b := range magic {
		if buf[i] != b {
			return false
		}
	}
	return true
}
