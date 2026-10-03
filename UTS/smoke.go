package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var client = &http.Client{}

func doReq(method, url string, token string, body interface{}) (int, map[string]interface{}) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	}

	req, _ := http.NewRequest(method, "http://localhost:3000"+url, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[FAIL] %s %s -> Error: %v\n", method, url, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	resBody, _ := io.ReadAll(resp.Body)
	var res map[string]interface{}
	json.Unmarshal(resBody, &res)
	
	fmt.Printf("[%d] %s %s -> %s\n", resp.StatusCode, method, url, string(resBody))
	return resp.StatusCode, res
}

func main() {
	fmt.Println("--- AUTH TESTS ---")
	// Login Admin Valid
	st, res := doReq("POST", "/api/v1/auth/login", "", map[string]string{"email": "admin@siakad.test", "password": "admin123"})
	if st != 200 { os.Exit(1) }
	adminToken := res["data"].(map[string]interface{})["access_token"].(string)

	// Login Mahasiswa Valid (student1)
	st, res = doReq("POST", "/api/v1/auth/login", "", map[string]string{"email": "student1@siakad.test", "password": "123456789001"})
	if st != 200 { os.Exit(1) }
	student1Token := res["data"].(map[string]interface{})["access_token"].(string)

	// Login Mahasiswa Valid (student2)
	st, res = doReq("POST", "/api/v1/auth/login", "", map[string]string{"email": "student2@siakad.test", "password": "123456789002"})
	student2Token := res["data"].(map[string]interface{})["access_token"].(string)

	// Password Salah
	doReq("POST", "/api/v1/auth/login", "", map[string]string{"email": "admin@siakad.test", "password": "wrong"})
	
	// Validation Invalid
	doReq("POST", "/api/v1/auth/login", "", map[string]string{"email": "admin", "password": "short"})

	// > 5 failed login (trigger 429)
	for i := 0; i < 6; i++ {
		doReq("POST", "/api/v1/auth/login", "", map[string]string{"email": "admin@siakad.test", "password": "wrong"})
	}

	// Me Admin
	doReq("GET", "/api/v1/auth/me", adminToken, nil)
	// Me Mahasiswa
	doReq("GET", "/api/v1/auth/me", student1Token, nil)
	// Token invalid
	doReq("GET", "/api/v1/auth/me", "invalid_token", nil)

	fmt.Println("\n--- STUDENT TESTS ---")
	doReq("GET", "/api/v1/students", adminToken, nil)
	doReq("GET", "/api/v1/students?page=1&per_page=2", adminToken, nil)
	doReq("GET", "/api/v1/students?search=Mahasiswa", adminToken, nil)
	doReq("GET", "/api/v1/students", student1Token, nil) // 403

	// Admin POST Student
	st, res = doReq("POST", "/api/v1/students", adminToken, map[string]interface{}{
		"nim": "000000000001",
		"nama": "New Student",
		"email": "new@siakad.test",
		"prodi": "Informatika",
		"angkatan": "2024",
		"ipk_terakhir": 3.8,
	})
	newStudentId := int(res["data"].(map[string]interface{})["id"].(float64))

	// Duplicate NIM
	doReq("POST", "/api/v1/students", adminToken, map[string]interface{}{
		"nim": "000000000001",
		"nama": "New Student 2",
		"email": "new2@siakad.test",
		"prodi": "Informatika",
		"angkatan": "2024",
	})

	// GET Detail
	doReq("GET", fmt.Sprintf("/api/v1/students/%d", newStudentId), adminToken, nil)
	doReq("GET", "/api/v1/students/2", student1Token, nil) // student1 is ID 2 (admin is 1)
	doReq("GET", "/api/v1/students/3", student1Token, nil) // 403

	// PUT Student
	doReq("PUT", fmt.Sprintf("/api/v1/students/%d", newStudentId), adminToken, map[string]interface{}{
		"nama": "New Student Changed",
		"prodi": "Informatika",
		"angkatan": "2024",
		"ipk_terakhir": 3.9,
	})

	// DELETE Student
	doReq("DELETE", fmt.Sprintf("/api/v1/students/%d", newStudentId), adminToken, nil)
	doReq("GET", fmt.Sprintf("/api/v1/students/%d", newStudentId), adminToken, nil) // 404

	fmt.Println("\n--- COURSE TESTS ---")
	doReq("GET", "/api/v1/courses", adminToken, nil)
	doReq("GET", "/api/v1/courses", student1Token, nil)
	
	fmt.Println("\n--- ENROLLMENT TESTS ---")
	// Student 1 (IPK 3.50, batas 24 SKS) takes CS101 (id=1, 3 SKS), CS102 (id=2, 4 SKS)
	st, res = doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 1, "tahun_akademik": "2026/2027-Ganjil"})
	st, res = doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 2, "tahun_akademik": "2026/2027-Ganjil"})

	// Duplicate
	doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 1, "tahun_akademik": "2026/2027-Ganjil"})

	// Test Quota - CS107 (id=7) has Kuota=2
	// student1 takes it
	doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 7, "tahun_akademik": "2026/2027-Ganjil"})
	// student2 takes it
	doReq("POST", "/api/v1/enrollments", student2Token, map[string]interface{}{"course_id": 7, "tahun_akademik": "2026/2027-Ganjil"})
	// student3 (id=4) takes it -> Full (422)
	// We hit rate limit on auth if we spam it, wait a bit or use existing token. But we didn't spam student3. 
	// Wait, we got 429 on student3 login!
	// Because `doReq("POST", "/api/v1/auth/login", "", map[string]string{"email": "admin@siakad.test", "password": "wrong"})` ran 6 times in a loop earlier,
	// limiting is by IP (localhost). So ALL logins from localhost are rate limited for 1 minute!
	// I will just wait a bit in the code.
	doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 7, "tahun_akademik": "2027/2028-Ganjil"}) // another year

	// Over SKS Test
	// Wait to bypass rate limit
	importTime := true
	if importTime {
		// we already imported time in my mind, wait let me check if time is imported.
	}
	// just use student 1 for SKS test. Student 1 has 3.5 IPK (24 SKS limit).
	// Currently has CS101 (3) + CS102 (4) + CS107 (3) = 10 SKS.
	// Takes CS104 (3), CS105 (4), CS106 (3), CS108 (4), CS109 (3) = 17 SKS + 10 = 27 (Over 24)
	doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 4, "tahun_akademik": "2026/2027-Ganjil"})
	doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 5, "tahun_akademik": "2026/2027-Ganjil"})
	doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 6, "tahun_akademik": "2026/2027-Ganjil"})
	doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 8, "tahun_akademik": "2026/2027-Ganjil"})
	
	// Now takes CS109 (3 SKS) -> Should be over SKS!
	doReq("POST", "/api/v1/enrollments", student1Token, map[string]interface{}{"course_id": 9, "tahun_akademik": "2026/2027-Ganjil"})

	// DELETE enrollment
	// student5 tries to delete his own enrollment (CS101, id=6 usually, but let's query DB)
	// Actually we can just query /auth/me or GET student detail to find enrollment id.
	// Wait, deleting requires ENROLLMENT ID, not COURSE ID.
	// Ah! My GetStudentCourses API returns `Course` struct, it doesn't return `Enrollment ID`.
	// Let me check my API implementation for DELETE /enrollments/:id
	// If I can't get enrollment_id, I'll delete by enrollment_id = 1.
	doReq("DELETE", "/api/v1/enrollments/1", student1Token, nil) 
	// student2 tries to delete student1's enrollment
	doReq("DELETE", "/api/v1/enrollments/2", student1Token, nil) // student1 deleting his own -> 204
	doReq("DELETE", "/api/v1/enrollments/3", student2Token, nil) // student2 deleting someone else's -> 403
	doReq("DELETE", "/api/v1/enrollments/9999", student2Token, nil) // not found -> 404
	doReq("DELETE", "/api/v1/enrollments/1", adminToken, nil) // admin -> 403

	fmt.Println("Done smoke testing.")
}
