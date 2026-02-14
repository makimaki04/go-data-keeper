package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/clientapp"
	"github.com/spf13/cobra"
)

var masterPassword, setItemType, file, userData, metaText, itemId string
var metaPairs []string

var types = []string{"login/pass", "text", "binary", "card"}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "command for set new item or update existing item using item_id",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !slices.Contains(types, setItemType) {
			return fmt.Errorf("unsupported type, allowed: %v", types)
		}

		data, err := GetData(file, userData, setItemType)
		if err != nil {
			return err
		}

		id, err := Uuid(itemId)
		if err != nil {
			return err
		}

		meta, err := parseMeta(metaPairs)
		if err != nil {
			return err
		}

		setOptions := clientapp.SetOptions{
			Meta:     meta,
			MetaText: metaText,
		}

		app := cmd.Context().Value(appKey{}).(*clientapp.App)

		return app.SetItem(masterPassword, data, setOptions, id, setItemType)
	},
}

func init() {
	setCmd.Flags().StringVar(&masterPassword, "password", "", "Master password")
	setCmd.Flags().StringVar(&setItemType, "type", "", "user data type: login/pass, text, binary, card")
	setCmd.Flags().StringVar(&file, "file", "", "binary data file path")
	setCmd.Flags().StringVar(&itemId, "id", "", "item id for update")
	setCmd.Flags().StringVar(&userData, "data", "", "user data")
	setCmd.Flags().StringVar(&metaText, "meta-text", "", "Free-form metadata text")
	setCmd.Flags().StringArrayVar(&metaPairs, "meta", nil, "Metadata key=value (repeatable)")
	_ = setCmd.MarkFlagRequired("password")
	_ = setCmd.MarkFlagRequired("type")
}

func parseMeta(pairs []string) (map[string]string, error) {
	const (
		maxPairs    = 50
		maxKeyLen   = 64
		maxValueLen = 4096
	)

	if len(pairs) > maxPairs {
		return nil, fmt.Errorf("too many --meta entries (max %d)", maxPairs)
	}

	out := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return nil, fmt.Errorf("meta must be key=value: %q", p)
		}

		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)

		if k == "" {
			return nil, fmt.Errorf("meta key can't be empty: %q", p)
		}

		if len(k) > maxKeyLen {
			return nil, fmt.Errorf("meta key too long (%d > %d): %q", len(k), maxKeyLen, k)
		}

		if len(v) > maxValueLen {
			return nil, fmt.Errorf("meta value too long (%d > %d) for key %q", len(v), maxValueLen, k)
		}

		out[k] = v
	}

	return out, nil
}

func Uuid(itemID string) (uuid.UUID, error) {
	if itemID != "" {
		id, err := uuid.Parse(itemID)
		if err != nil {
			return uuid.Nil, err
		}

		return id, nil
	}

	return uuid.New(), nil
}

func GetData(file string, userData string, itemType string) ([]byte, error) {
	if file != "" && userData != "" {
		return nil, fmt.Errorf("use either --file or --data, not both")
	}

	if itemType == "binary" {
		if file == "" {
			return nil, fmt.Errorf("binary requires --file")
		}

		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	if userData == "" {
		return nil, fmt.Errorf("type %s requires --data", itemType)
	}

	return []byte(userData), nil
}
