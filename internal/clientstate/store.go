package clientstate

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/makimaki04/go-data-keeper.git/pkg/contract"
	"go.uber.org/zap"
)

// FileStore identifies a file used for persistent storage.
type FileStore struct {
	path string
}

// Store persists client state and vault data on disk.
type Store struct {
	stateFile FileStore
	vaultFile FileStore
	logger    *zap.SugaredLogger
}

// NewStore creates a Store and ensures that its parent directories exist.
// NewStore returns an error if the store directories can't be created.
func NewStore(statePath string, vaultPath string, logger *zap.SugaredLogger) (*Store, error) {
	logger = logger.With("component", "store")

	stateDir := filepath.Dir(statePath)
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		logger.Errorw("could't create state dir", "op", "new_state")
		return &Store{}, err
	}

	vaultDir := filepath.Dir(vaultPath)
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		logger.Errorw("could't create vault dir", "op", "new_state")
		return &Store{}, err
	}

	return &Store{
		stateFile: FileStore{
			path: statePath,
		},
		vaultFile: FileStore{
			path: vaultPath,
		},
		logger: logger,
	}, nil
}

// SaveState writes the provided state to disk.
// SaveState returns an error if the state can't be encoded or written.
func (s *Store) SaveState(data State) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		s.logger.Errorw("data json.marshal error", "op", "state.save_state")
		return err
	}

	if err := atomicWriteFile(s.stateFile.path, jsonData, 0600); err != nil {
		s.logger.Errorw("atomic write file error",
			"op", "state.save_state",
			"file_path", s.stateFile.path,
			"err", err,
		)
		return err
	}

	return nil
}

// LoadState loads the state from disk.
// LoadState returns an empty State if the state file does not exist.
// LoadState returns an error if the file can't be read or decoded.
func (s *Store) LoadState() (State, error) {
	var state State

	jsonData, err := os.ReadFile(s.stateFile.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Infow("state file doesn't exist",
				"op", "state.load_state",
				"file_path", s.stateFile.path,
			)
			return State{}, nil
		}

		s.logger.Errorw("os.readFile error",
			"op", "state.load_state",
			"file_path", s.stateFile.path,
			"err", err,
		)
		return State{}, err
	}

	if err := json.Unmarshal(jsonData, &state); err != nil {
		s.logger.Errorw("json unmarshal error",
			"op", "state.load_state",
			"err", err,
		)
		return State{}, err
	}

	return state, nil
}

// SaveVault writes the provided vault to disk.
// SaveVault returns an error if the vault can't be encoded or written.
func (s *Store) SaveVault(vault Vault) error {
	jsonData, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		s.logger.Errorw("data json.marshal error", "op", "vault.save_vault")
		return err
	}

	if err := atomicWriteFile(s.vaultFile.path, jsonData, 0600); err != nil {
		s.logger.Errorw("atomic write file error",
			"op", "vault.save_vault",
			"file_path", s.vaultFile.path,
			"err", err,
		)
		return err
	}

	return nil
}

// LoadVault loads the vault from disk.
// LoadVault returns an empty vault if the vault file does not exist.
// LoadVault returns an error if the file can't be read or decoded.
func (s *Store) LoadVault() (Vault, error) {
	var vault Vault

	jsonData, err := os.ReadFile(s.vaultFile.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Infow("vault file doesn't exist",
				"op", "vault.load_vault",
				"file_path", s.vaultFile.path,
			)
			return Vault{
				Store: make(map[string]contract.ItemDTO),
			}, nil
		}

		s.logger.Errorw("os.readFile error",
			"op", "vault.load_vault",
			"file_path", s.vaultFile.path,
			"err", err,
		)
		return Vault{}, err
	}

	if err := json.Unmarshal(jsonData, &vault); err != nil {
		s.logger.Errorw("json unmarshal error",
			"op", "vault.load_vault",
			"err", err,
		)
		return Vault{}, err
	}

	if vault.Store == nil {
		vault.Store = make(map[string]contract.ItemDTO)
	}

	return vault, nil
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}

	tmpName := tmp.Name()

	defer func() {
		_ = tmp.Close()
	}()

	if err := tmp.Chmod(perm); err != nil {
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		return err
	}

	if err := tmp.Sync(); err != nil {
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	return nil
}

// WipeVault clears the vault on disk.
// WipeVault returns an error if the vault can't be encoded or written.
func (s *Store) WipeVault() error {
	var vault Vault
	vault.Store = make(map[string]contract.ItemDTO)

	jsonData, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		s.logger.Errorw("data json.marshal error", "op", "vault.wipe_vault")
		return err
	}

	if err := atomicWriteFile(s.vaultFile.path, jsonData, 0600); err != nil {
		s.logger.Errorw("atomic write file error",
			"op", "vault.wipe_vault",
			"file_path", s.vaultFile.path,
			"err", err,
		)
		return err
	}

	return nil

}
