package modelsPal

type Classification struct {
	Id				string				`json:"id"`
	Identifier		string				`json:"identifier"`
	Name			string				`json:"name"`
	SortIndex		int					`json:"sortIndex"`
	IsRoot			bool				`json:"isRoot"`
	ParentId		string				`json:"parentId"`
}
