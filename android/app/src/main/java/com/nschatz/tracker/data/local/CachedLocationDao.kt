package com.nschatz.tracker.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query

@Dao
interface CachedLocationDao {

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(entity: CachedLocationEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsertAll(entities: List<CachedLocationEntity>)

    @Query("SELECT * FROM cached_locations")
    suspend fun getAll(): List<CachedLocationEntity>

    @Query("DELETE FROM cached_locations")
    suspend fun clear()
}
