package events

import "strconv"

// old code for backwards compatibility

type keyval struct {
	Key, Val string
}

type encodableMetric struct {
	TimestampUnix int64
	Data          []keyval
}

func transform(m *encodableMetric) (*Deployment, error) {
	// Default values are created here
	d := &Deployment{}
	var err error
	for _, kv := range m.Data {
		k := kv.Key
		v := kv.Val
		switch k {
		case "StartTime":
			d.StartTime, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
		case "EndTime":
			d.EndTime, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
		case "Username":
			d.Username = v
		case "ModelName":
			d.ModelName = v
		case "ModelLang":
			d.ModelLang = v
		case "ModelDeps":
			d.ModelDeps = v
		case "ModelVer":
			yi64, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			d.ModelVer = yi64
		case "ModelSize":
			yi64, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			d.ModelSize = yi64
		default:
		}
	}
	return d, nil
}
