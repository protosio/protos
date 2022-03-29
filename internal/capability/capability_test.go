package capability

// import (
// 	"testing"

// 	"github.com/golang/mock/gomock"
// )

// func TestCapabilityManager(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	cm := CreateManager()
// 	testCap := cm.New("test")

// 	t.Run("New", func(t *testing.T) {
// 		if cm.allCapabilities[len(cm.allCapabilities)-1] != testCap {
// 			t.Error("The newly created test capability should be in the AllCapabilities list")
// 		}
// 	})

// 	t.Run("Validate", func(t *testing.T) {
// 		if cm.Validate(testCap, "test") != true {
// 			t.Error("Validate: test capability should validate against its name")
// 		}
// 		newCap := cm.New("new")
// 		testCap.SetParent(newCap)
// 		if cm.Validate(testCap, "new") != true {
// 			t.Error("Validate: test capability should validate against its parent capability")
// 		}

// 		testCap.Parent = nil
// 		if cm.Validate(testCap, "wrongCap") != false {
// 			t.Error("Validate should return false when an incorrect capability name is provided")
// 		}
// 	})

// 	t.Run("Set and GetMethodCap", func(t *testing.T) {
// 		cm.SetMethodCap("testMethod", testCap)

// 		func() {
// 			defer func() {
// 				r := recover()
// 				if r == nil {
// 					t.Errorf("SetMethodCaps hould panic when a method already has a capability")
// 				}
// 			}()
// 			cm.SetMethodCap("testMethod", testCap)
// 		}()

// 		cap, err := cm.GetMethodCap("testMethod")
// 		if err != nil {
// 			t.Errorf("GetMethodCap should not return an error: %s", err.Error())
// 		}
// 		if cap != testCap {
// 			t.Errorf("GetMethodCap returned an incorrect capability: %p vs %p", cap, testCap)
// 		}

// 		_, err = cm.GetMethodCap("wrongMethod")
// 		if err == nil {
// 			t.Error("GetMethodCap should return an error when a method does not have a capability associated with it")
// 		}
// 	})

// 	t.Run("GetByName", func(t *testing.T) {
// 		_, err := cm.GetByName("wrongCap")
// 		if err == nil {
// 			t.Error("GetByName should return an error when a non-existent capability is provided")
// 		}

// 		cap, err := cm.GetByName("test")
// 		if err != nil {
// 			t.Errorf("GetByName should NOT return an error: %s", err.Error())
// 		}
// 		if cap != testCap {
// 			t.Errorf("GetByName returned an incorrect capability: %p vs %p", cap, testCap)
// 		}
// 	})

// 	t.Run("GetOrPanic", func(t *testing.T) {
// 		func() {
// 			defer func() {
// 				r := recover()
// 				if r == nil {
// 					t.Errorf("GetOrPanic should panic if a non-existent capability is requested")
// 				}
// 			}()
// 			cm.GetOrPanic("wrongCap")
// 		}()

// 		cap := cm.GetOrPanic("test")
// 		if cap != testCap {
// 			t.Errorf("GetOrPanic returned an incorrect capability: %p vs %p", cap, testCap)
// 		}
// 	})

// 	t.Run("ClearAll", func(t *testing.T) {
// 		if len(cm.capMap) == 0 {
// 			t.Error("ClearAll test should have a capMap with more than one capability when this test is run")
// 		}
// 		cm.ClearAll()
// 		if len(cm.capMap) != 0 {
// 			t.Errorf("ClearAll should lead to the capMap having 0 elements, but it has %d", len(cm.capMap))
// 		}
// 	})
// }
