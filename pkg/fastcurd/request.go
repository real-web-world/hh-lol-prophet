package fastcurd

const (
	SceneDefault = "default"
	NotLimit     = 0
)

type (
	NullableDetailData struct {
		ID    *int                   `json:"id" binding:"omitempty"`
		Scene string                 `json:"scene" binding:"omitempty"`
		Extra map[string]interface{} `json:"extra" binding:"omitempty"`
	}
	IDData struct {
		ID int `json:"id" binding:"required"`
	}
	DetailData struct {
		ID    int                    `json:"id" binding:"required"`
		Scene string                 `json:"scene" binding:""`
		Extra map[string]interface{} `json:"extra" binding:""`
	}
	DelData struct {
		IDs []int `json:"ids" binding:"required,min=1"`
	}
	ListData struct {
		Page   int                    `json:"page" binding:"omitempty,required,min=0"`
		Limit  int                    `json:"limit" binding:"omitempty,required,min=0,max=50"`
		Filter Filter                 `json:"filter" binding:""`
		Order  map[string]string      `json:"order" binding:""`
		Extra  map[string]interface{} `json:"extra" binding:""`
	}
	FullLimitListData struct {
		ListData
		Limit int `json:"limit" binding:"omitempty,required,min=0"`
	}
	SearchData struct {
		Search string `json:"search" binding:"required"`
		ListData
	}
)

func (p *ListData) GetScene() string {
	var scene interface{}
	var ok bool
	if scene, ok = p.Extra["scene"]; !ok {
		scene = SceneDefault
	}
	return scene.(string)
}
func (d *DetailData) GetScene() string {
	scene := d.Scene
	if scene == "" {
		scene = SceneDefault
	}
	return scene
}
func (d *NullableDetailData) GetScene() string {
	scene := d.Scene
	if scene == "" {
		scene = SceneDefault
	}
	return scene
}
