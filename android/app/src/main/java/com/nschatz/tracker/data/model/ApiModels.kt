package com.nschatz.tracker.data.model

import com.google.gson.annotations.SerializedName

data class RegisterRequest(
    @SerializedName("email") val email: String,
    @SerializedName("password") val password: String,
    @SerializedName("display_name") val displayName: String
)

data class LoginRequest(
    @SerializedName("email") val email: String,
    @SerializedName("password") val password: String
)

data class AuthResponse(
    @SerializedName("token") val token: String,
    @SerializedName("user") val user: UserInfo
)

data class UserInfo(
    @SerializedName("id") val id: String,
    @SerializedName("email") val email: String,
    @SerializedName("display_name") val displayName: String
)

data class LocationBatch(
    @SerializedName("circle_id") val circleId: String,
    @SerializedName("points") val points: List<LocationPoint>
)

data class LocationPoint(
    @SerializedName("lat") val lat: Double,
    @SerializedName("lng") val lng: Double,
    @SerializedName("speed") val speed: Float?,
    @SerializedName("battery_level") val batteryLevel: Int?,
    @SerializedName("accuracy") val accuracy: Float?,
    @SerializedName("recorded_at") val recordedAt: String
)

data class MemberLocation(
    @SerializedName("user_id") val userId: String,
    @SerializedName("display_name") val displayName: String,
    @SerializedName("lat") val lat: Double,
    @SerializedName("lng") val lng: Double,
    @SerializedName("battery_level") val batteryLevel: Int?,
    @SerializedName("recorded_at") val recordedAt: String
)

data class CreateCircleRequest(
    @SerializedName("name") val name: String
)

data class JoinCircleRequest(
    @SerializedName("invite_code") val inviteCode: String
)

data class Circle(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
    @SerializedName("invite_code") val inviteCode: String,
    @SerializedName("owner_id") val ownerId: String
)

data class CircleMember(
    @SerializedName("user_id") val userId: String,
    @SerializedName("display_name") val displayName: String,
    @SerializedName("role") val role: String
)

data class CreateGeofenceRequest(
    @SerializedName("circle_id") val circleId: String,
    @SerializedName("name") val name: String,
    @SerializedName("lat") val lat: Double,
    @SerializedName("lng") val lng: Double,
    @SerializedName("radius_meters") val radiusMeters: Double
)

data class UpdateGeofenceRequest(
    @SerializedName("name") val name: String?,
    @SerializedName("radius_meters") val radiusMeters: Double?
)

data class Geofence(
    @SerializedName("id") val id: String,
    @SerializedName("circle_id") val circleId: String,
    @SerializedName("name") val name: String,
    @SerializedName("lat") val lat: Double,
    @SerializedName("lng") val lng: Double,
    @SerializedName("radius_meters") val radiusMeters: Double
)
