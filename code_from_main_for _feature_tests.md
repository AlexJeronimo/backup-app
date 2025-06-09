//--- CRUD for user testing ---
	userRepo := database.NewUserRepo(db)

	log.Println("\n--- Testing CRUD for users ---")

	log.Println("Create user admin...")
	adminUser, err := userRepo.CreateUser("admin", "supersecurepassword")
	if err != nil {
		log.Printf("User 'admin' creation error: %v", err)
	} else {
		log.Printf("User 'admin' successfully created: %v", adminUser)
	}

	log.Println("Create user testuser...")
	testUser, err := userRepo.CreateUser("testuser", "supersecurepassword")
	if err != nil {
		log.Printf("User 'adtestuserin' creation error: %v", err)
	} else {
		log.Printf("User 'testuser' successfully created: %v", testUser)
	}

	log.Println("Creating existing user 'admin', expecting to reveive error...")
	_, err = userRepo.CreateUser("admin", "anotherpassword")
	if err != nil {
		log.Printf("Success: Received expected error during user creation: %v", err)
	} else {
		log.Printf("Error: created existing user, not expected.")
	}

	log.Println("Search for user 'admin'...")
	foundAdmin, err := userRepo.GetUserByUsername("admin")
	if err != nil {
		log.Printf("Search user 'admin' error: %v", err)
	} else {
		log.Printf("Found user 'admin': %+v", foundAdmin)
	}

	log.Println("Search for nonexistent user....")
	_, err = userRepo.GetUserByUsername("nonexistent")
	if err != nil {
		log.Printf("Success: Received expected error during nonexistent user search: %v", err)
	} else {
		log.Println("Error: found nonexistent user, not expected.")
	}

	log.Println("User 'admin' authentication with right password...")
	authAdmin, err := userRepo.AuthenticateUser("admin", "supersecurepassword")
	if err != nil {
		log.Printf("Authentication error for 'admin': %v", err)
	} else {
		log.Printf("User 'admin' authentication successfull: %+v", authAdmin)
	}

	log.Println("User 'admin' authentication with wrong password, expected error...")
	_, err = userRepo.AuthenticateUser("admin", "wrongsupersecurepassword")
	if err != nil {
		log.Printf("Successfull: Received expected authentication error for 'admin' with wrong password: %v", err)
	} else {
		log.Println("Error: User 'admin' authentication with wrong password successful, not expected.")
	}

	if testUser != nil {
		log.Println("Update password for user 'testuser'...")
		err = userRepo.UpdateUser(testUser, "newtestpassword")
		if err != nil {
			log.Printf("Error update 'testuser': %v", err)
		} else {
			log.Printf("Password for 'testuser' successfully updated. New hash: %s...", testUser.PasswordHash[:5])
			_, err = userRepo.AuthenticateUser("testuser", "newtestpassword")
			if err != nil {
				log.Printf("Authentication error for 'testuser' with new password: %v", err)
			} else {
				log.Println("Authentication successfull for 'testuser' with new password.")
			}
		}
	}

	log.Println("Get All users...")
	allUsers, err := userRepo.GetAllUsers()
	if err != nil {
		log.Printf("All user getting error: %v", err)
	} else {
		log.Printf("Found %d users:", len(allUsers))
		for _, u := range allUsers {
			createdAtStr := "N/A"
			if u.CreatedAt.Valid {
				createdAtStr = u.CreatedAt.Time.Format("2006-01-02 15:04:05")
			}
			log.Printf(" - ID: %d, Username: %s, CreatedAt: %s", u.ID, u.Username, createdAtStr)
		}
	}

	if testUser != nil {
		log.Printf("Delet user 'testuser' (ID: %d)...", testUser.ID)
		err = userRepo.DeleteUser(testUser.ID)
		if err != nil {
			log.Printf("User delete error for 'testuser': %v", err)
		} else {
			log.Println("User 'testuser' was deleted successfully.")
		}
	}

	log.Println("Check 'testuser' was deleted...")
	_, err = userRepo.GetUserByUsername("testuser")
	if err != nil && (err.Error() == fmt.Sprintf("user '%s' not found", "testuser")) {
		log.Println("Success: User 'testuser' not found after deleting.")
	} else if err != nil {
		log.Printf("Error during user deletion checking for 'testuser': %v", err)
	} else {
		log.Println("Error: User 'testuser' was not deleted.")
	}

	log.Println("---End of CRUD testing for user---")

	//--- End of CRUD testing for user ---

	//--- Testing CRUD for backup tasks ---

	jobRepo := database.NewJobRepo(db)

	log.Println("\n--- Testing CRUD for backup tasks ---")

	log.Println("Creates backup task 'Daily Documents backup'...")
	job1, err := jobRepo.CreateJob("Daily Document Backup", "C:\\Users\\user\\Documents", "E:\\BackupData\\Documents", "daily", true)
	if err != nil {
		log.Printf("Error creating task 'Daily Documents Backup': %v", err)
	} else {
		createdAtStr := "N/A"
		if job1.CreatedAt.Valid {
			createdAtStr = job1.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
		}
		updatedAtStr := "N/A"
		if job1.UpdatedAt.Valid {
			updatedAtStr = job1.UpdatedAt.Time.Format("2006-01-02 15:04:05.000")
		}

		log.Printf("Backup task 'Daily Documents Backup' successfully created: ID: %d, Name: %s, Schedule: %s, Created: %s, Updated: %s",
			job1.ID, job1.Name, job1.Schedule, createdAtStr, updatedAtStr)
	}

	log.Println("Creates backup task 'Weekly Photos Backup'...")
	job2, err := jobRepo.CreateJob("Weekly Photos Backup", "C:\\Users\\user\\Pictures", "E:\\BackupData\\Photos", "weekly", true)
	if err != nil {
		log.Printf("Error creating task 'Weekly Photos Backup': %v", err)
	} else {
		createdAtStr := "N/A"
		if job2.CreatedAt.Valid {
			createdAtStr = job2.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
		}
		updatedAtStr := "N/A"
		if job2.UpdatedAt.Valid {
			updatedAtStr = job2.UpdatedAt.Time.Format("2006-01-02 15:04:05.000")
		}
		log.Printf("Backup task 'Weekly Photos Backup' successfully created: ID: %d, Name: %s, Schedule: %s, Created: %s, Updated: %s",
			job2.ID, job2.Name, job2.Schedule, createdAtStr, updatedAtStr)
	}

	//Try to create existing task

	log.Println("Trying to create backup task 'Daily Documents backup' which exists")
	_, err = jobRepo.CreateJob("Daily Document Backup", "C:\\Users\\user\\Documents", "E:\\BackupData\\Documents", "daily", true)
	if err != nil {
		log.Printf("Success: got expected error during existing task creating: %v", err)
	} else {
		log.Printf("Error: created existing task, not expected.")
	}

	//Get task by ID
	if job1 != nil {
		log.Printf("Search for task by ID %d...", job1.ID)
		foundJob, err := jobRepo.GetJobByID(job1.ID)
		if err != nil {
			log.Printf("Error search task by ID %d: %v", job1.ID, err)
		} else {
			createdAtStr := "N/A"
			if foundJob.CreatedAt.Valid {
				createdAtStr = foundJob.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			updatedAtStr := "N/A"
			if foundJob.UpdatedAt.Valid {
				updatedAtStr = foundJob.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			log.Printf("Found task by ID %d: Name: %s, Source: %s, IsActive: %t, Created: %s, Updated: %s",
				foundJob.ID, foundJob.Name, foundJob.SourcePath, foundJob.IsActive, createdAtStr, updatedAtStr)
		}
	}

	//Get task by name
	if job1 != nil {
		log.Println("Search for task 'Weekly Photos Backup'...")
		foundJobByName, err := jobRepo.GetJobByName("Weekly Photos Backup")
		if err != nil {
			log.Printf("Error search task 'Weekly Photos Backup': %v", err)
		} else {
			createdAtStr := "N/A"
			if foundJobByName.CreatedAt.Valid {
				createdAtStr = foundJobByName.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			updatedAtStr := "N/A"
			if foundJobByName.UpdatedAt.Valid {
				updatedAtStr = foundJobByName.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			log.Printf("Found task 'Weekly Photos Backup': ID %d, Name: %s, Source: %s, IsActive: %t, Created: %s, Updated: %s",
				foundJobByName.ID, foundJobByName.Name, foundJobByName.SourcePath, foundJobByName.IsActive, createdAtStr, updatedAtStr)
		}
	}

	//Update task
	if job1 != nil {
		log.Printf("Update task 'Daily Documents Backup' (ID %d) - change schedule and do inactive...", job1.ID)
		job1.Schedule = "monthly"
		job1.IsActive = false
		err = jobRepo.UpdateJob(job1)
		if err != nil {
			log.Printf("Error during update task 'Daily Documents Backup': %v", err)
		} else {
			updatedJob, _ := jobRepo.GetJobByID(job1.ID)
			updatedAtStr := "N/A"
			if updatedJob.UpdatedAt.Valid {
				updatedAtStr = updatedJob.UpdatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			log.Printf("Task 'Daily Documents Backup' successfully updated. New Schedule: %s, IsActive: %t, Updated: %s",
				updatedJob.Schedule, updatedJob.IsActive, updatedAtStr)
		}
	}

	//Get all tasks
	log.Println("Get all backup tasks...")
	allJobs, err := jobRepo.GetAllJobs()
	if err != nil {
		log.Printf("Error getting all backup tasks: %v", err)
	} else {
		log.Printf("Found %d backup tasks:", len(allJobs))
		for _, j := range allJobs {
			createdAtStr := "N/A"
			if j.CreatedAt.Valid {
				createdAtStr = j.CreatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			updatedAtStr := "N/A"
			if j.UpdatedAt.Valid {
				updatedAtStr = j.UpdatedAt.Time.Format("2006-01-02 15:04:05.000")
			}
			log.Printf(" - ID: %d, Name: %s, Active: %t, Created: %s, Updated: %s",
				j.ID, j.Name, j.IsActive, createdAtStr, updatedAtStr)
		}
	}

	//Delete task
	if job2 != nil {
		log.Printf("Deleting task 'Weekly Photos Backup' (ID: %d)...", job2.ID)
	}
	err = jobRepo.DeleteJob(job2.ID)
	if err != nil {
		log.Printf("Error deleting task 'Weekly Photos Backup': %v", err)
	} else {
		log.Println("Task 'Weekly Photos Backup' successfully deleted.")
	}

	log.Println("Check that 'Weekly Photos Backup' was deleted...")
	_, err = jobRepo.GetJobByName("Weekly Photos Backup")
	if err != nil && (err.Error() == fmt.Sprintf("backup task with '%s' not found", "Weekly Photos Backup")) {
		log.Println("Success: task 'Weekly Photos Backup' not found after deleting")
	} else if err != nil {
		log.Printf("Error during checking deleting 'Weekly Photos Backup': %v", err)
	} else {
		log.Println("Error: task 'Weekly Photos Backup' not deleted.")
	}

	log.Println("--- end of testing CRUD for tasks ---")
	//--- End of CRUD testing for backup tasks ---

	//--- Backup Test ---
	sourceFile := "E:\\test.txt"

	destFile := filepath.Join(cfg.BackupDir, "test_backup", filepath.Base(sourceFile))

	log.Printf("Backup try: %s -> %s\n", sourceFile, destFile)
	err = backup.CopyFiles(sourceFile, destFile)
	if err != nil {
		log.Printf("Local backup error: %v", err)
	} else {
		log.Printf("Local backup successfully completed for test file.")
	}

	//--- End of Backup Test ---

	//---Extended backup test ----

	log.Println("\n--- Test exetended backup logic ---")

	testSourceDir := filepath.Join("E:", "BackupTest_Source")
	testDestDir := filepath.Join(cfg.BackupDir, "BackupTest_Dest")

	os.RemoveAll(testSourceDir) //delete old if exist
	os.RemoveAll(testDestDir)   //delete old if exist
	os.MkdirAll(filepath.Join(testSourceDir, "sub1"), 0755)
	os.MkdirAll(filepath.Join(testSourceDir, "sub2"), 0755)
	os.WriteFile(filepath.Join(testSourceDir, "file1.txt"), []byte("This is file 1."), 0644)
	os.WriteFile(filepath.Join(testSourceDir, "sub1", "file2.txt"), []byte("This is file 2 in sub1."), 0644)
	os.WriteFile(filepath.Join(testSourceDir, "sub2", "file3.log"), []byte("LOg content."), 0644)

	log.Printf("Test directories prepared: %s", testSourceDir)

	log.Println("\n--- First launch CopyDir (full backup) ---")
	var totalCopiedFiles int
	var totalCopiedBytes int64
	backupStartTime := time.Now()
	err = backup.CopyDir(testSourceDir, testDestDir, &totalCopiedFiles, &totalCopiedBytes)
	if err != nil {
		log.Printf("Error during first backup launch: %v", err)
	} else {
		log.Printf("First backup completed successfully. Copied files: %d, Size: %d bytes", totalCopiedFiles, totalCopiedBytes) 
	}
	log.Printf("Time of first backup: %s", time.Since(backupStartTime))

	log.Println("\n--- Second launch CopyDir (expecting file skipping) ----")
	var totalCopiedFiles2 int
	var totalCopiedBytes2 int64
	backupStartTime = time.Now()
	err = backup.CopyDir(testSourceDir, testDestDir, &totalCopiedFiles2, &totalCopiedBytes2)
	if err != nil {
		log.Printf("Error during second backup launch: %v", err)
	} else {
		log.Printf("Second backup completed successfully. Copied files: %d, Sizee: %d bytes", totalCopiedFiles2, totalCopiedBytes2)
		if totalCopiedFiles2 == 0 && totalCopiedBytes2 == 0 {
			log.Println("Success: no one file was copied, as expected (optimization work)")
		} else {
			log.Println("Error: Was copied files during second launch, although were no any changes.")
		}
	}
	log.Printf("Second launch execution time: %s", time.Since(backupStartTime))

	log.Println("\n--- Change file in source folder and third launch of backup ---")
	modifiedFilePath := filepath.Join(testSourceDir, "file1.txt")
	os.WriteFile(modifiedFilePath, []byte("This is file 1, updated content!"), 0644)

	log.Printf("Changed file: %s", modifiedFilePath)
	time.Sleep(1 * time.Second)

	var totalCopiedFiles3 int
	var totalCopiedBytes3 int64
	backupStartTime = time.Now()
	err = backup.CopyDir(testSourceDir, testDestDir, &totalCopiedFiles3, &totalCopiedBytes3)
	if err != nil {
		log.Printf("Error during third backup launch: %v", err)
	} else {
		log.Printf("Third backup successfully completed. Copied files: %d, Size: %d bytes", totalCopiedFiles3, totalCopiedBytes3)
		if totalCopiedFiles3 == 1 {
			log.Println("Success: Copied exact 1 file, as expected (update changed file).")
		} else {
			log.Printf("Error: Copied %d files, expected 1", totalCopiedFiles3)
		}
	}
	log.Printf("Third backup execution time: %s", time.Since(backupStartTime))

	log.Println("\n--- Testing GetDirSize ---")
	sourceSize, err := backup.GetDirSIze(testSourceDir)
	if err != nil {
		log.Printf("Error during getting size of source directory: %v", err)
	} else {
		log.Printf("Size of source directory '%s' is: %d bytes", testSourceDir, sourceSize)
	}

	destSize, err := backup.GetDirSIze(testDestDir)
	if err != nil {
		log.Printf("Error during getting size of destination directory: %v", err)
	} else {
		log.Printf("Size of destination directory '%s' is: %d bytes", testDestDir, destSize)
	}

	//--- End of Extended backup test