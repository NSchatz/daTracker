package com.nschatz.tracker.data.local

import androidx.room.ColumnInfo
import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "pending_locations")
data class PendingLocationEntity(
    @PrimaryKey(autoGenerate = true) val id: Long = 0,
    @ColumnInfo(name = "lat") val lat: Double,
    @ColumnInfo(name = "lng") val lng: Double,
    @ColumnInfo(name = "speed") val speed: Float?,
    @ColumnInfo(name = "battery_level") val batteryLevel: Int?,
    @ColumnInfo(name = "accuracy") val accuracy: Float?,
    @ColumnInfo(name = "recorded_at") val recordedAt: String
)

@Entity(tableName = "cached_locations")
data class CachedLocationEntity(
    @PrimaryKey @ColumnInfo(name = "order_id") val orderId: String,
    @ColumnInfo(name = "user_id") val userId: String,
    @ColumnInfo(name = "display_name") val displayName: String,
    @ColumnInfo(name = "lat") val lat: Double,
    @ColumnInfo(name = "lng") val lng: Double,
    @ColumnInfo(name = "battery_level") val batteryLevel: Int?,
    @ColumnInfo(name = "recorded_at") val recordedAt: String,
    @ColumnInfo(name = "updated_at") val updatedAt: String
)
