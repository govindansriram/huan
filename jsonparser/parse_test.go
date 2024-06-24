package jsonparser

import (
	"testing"
)

func TestAttemptConversion(t *testing.T) {

	t.Run("test valid array", func(t *testing.T) {
		arr := `[{"one": 1}, {"two": 2}]`

		err, data := AttemptConversion(arr)

		if err != nil {
			t.Error(err)
		}

		if len(data) != 2 {
			t.Errorf("could not unmarshall data properly")
		}
	})

	t.Run("test valid map", func(t *testing.T) {
		arr := `{"two": 2}`

		err, data := AttemptConversion(arr)

		if err != nil {
			t.Error(err)
		}

		if len(data) != 1 {
			t.Errorf("could not unmarshall data properly")
		}
	})
}

func TestToJson(t *testing.T) {

	t.Run("test valid json in sentence", func(t *testing.T) {
		arr := ` asdasds dassad dasdsa das [{"one": 1}, {"two": 2}] asdasdadsaasdasd`

		data := ToJson(arr)

		if len(data) != 2 {
			t.Errorf("could not unmarshall data properly")
		}
	})

	t.Run("test broken json in sentence", func(t *testing.T) {
		arr := ` asdasds [{"one": 1} dasdsa da, {"two": 2}] asdasdadsaasdasd`

		data := ToJson(arr)

		if len(data) != 2 {
			t.Errorf("could not unmarshall data properly")
		}
	})

	t.Run("test broken map in first sentence", func(t *testing.T) {
		arr := ` asdasds [}{"one":} 1}}} dasdsa da, {"two": 2}] asdasdadsaasdasd`

		data := ToJson(arr)

		if len(data) != 1 {
			t.Errorf("could not unmarshall data properly")
		}
	})

	t.Run("test completely broken map", func(t *testing.T) {
		arr := ` asdasds [{{"one": 1} dasdsa da, {"two": 2}] asdasdadsaasdasd`

		data := ToJson(arr)

		if len(data) != 0 {
			t.Errorf("could not unmarshall data properly")
		}
	})
}
