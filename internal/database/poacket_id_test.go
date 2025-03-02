package database

import "testing"

func TestPacketID(t *testing.T) {
	mgr := NewPacketIDManager()
	mgr2 := NewPacketIDManager()
	if mgr != mgr2 {
		t.Fatal("manager does not match")
	}

	// 测试分配
	id1 := mgr.NextID()
	if id1 != 1 {
		t.Fatalf("Expected 1, got %d", id1)
	}

	// 测试释放与复用
	mgr.ReleaseID(id1)
	id2 := mgr.NextID()
	if id2 != 1 {
		t.Fatalf("Expected 1 after release, got %d", id2)
	}

	// 测试溢出
	mgr.currentID = 65535
	id3 := mgr.NextID()
	if id3 != 65535 {
		t.Fatalf("Expected 65535, got %d", id3)
	}
	id4 := mgr.NextID()
	if id4 != 1 {
		t.Fatalf("Expected 1 after overflow, got %d", id4)
	}
}
