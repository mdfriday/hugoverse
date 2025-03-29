package repository

type Repository interface {
	PutContent(ci any, data []byte) error
	NewContent(ci any, data []byte) error

	AllContent(namespace string) [][]byte
	ContentByPrefix(namespace, prefix string) ([][]byte, error)
	GetContent(namespace string, id string) ([]byte, error)
	GetRandomContent(namespace string) ([]byte, error)
	DeleteContent(namespace string, id string, slug string, hash string) error

	NextContentId(ns string) (uint64, error)
	CheckSlugForDuplicate(namespace string, slug string) (string, error)
	GetIdByHash(namespace string, hash string) ([]byte, error)

	PutSortedContent(namespace string, m map[string][]byte) error

	UserDataDir() string
	AdminDataDir() string
}
