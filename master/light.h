//
// Copyright (C) 2012-2014 Light Transport Entertainment Inc.
//
#ifndef __LTE_LIGHT_H__
#define __LTE_LIGHT_H__

typedef enum
{
    LIGHT_QUAD,           // Quad area light
    LIGHT_SPHERE,         // Spherical area light
    LIGHT_MESH,           // Mesh light
    LIGHT_POINT,
    LIGHT_DIRECTIONAL,
    LIGHT_SPOT,
    LIGHT_SUN,
} LightType;

typedef enum
{
    LIGHT_DECAY_NONE,
    LIGHT_DECAY_LINEAR,
    LIGHT_DECAY_QUADRATIC,
    LIGHT_DECAY_CUBIC
} LightDecay;

typedef enum
{
    LIGHT_LUMEN,          // [lm]
    LIGHT_LUX,            // [lx]
    LIGHT_CANDERA,        // [cd/m^2]
} LightUnit;

typedef struct
{
    LightType    type;
    LightUnit    unit;
    int          enabled;     // 1 = light is on, 0 = light is off
    int          visible;
    int          material_id;
    int          photometric; // 1 = photometric mode, 0 = color mode.
    float        intensity;
    float        intensity_rgb[3];
    float        intensity_scale;
    float        temperature;   // [K]

    // World coordinate parameter(after transformed by light matrix);
    float        position[3];       // Position in world coordinate.
    float        normal[3];         // Normal in world coordiante.

    // Local coordinate paramter(before light matrix)
    float        position_local[3]; // Position in local coordiate.
    float        normal_local[3];   // Normal in local coordiate.

    // params
    float        shadow_bias;
    int          cast_shadow; 
    int          double_sided;

    // For directional and spot
    //float        direction[3];    // Use normal instead.
    LightDecay   decay;
    float        cone_angle;        // in degree
    float        penumbla_angle;    // in degree
    float        dropoff;           // [0.0, 1.0]

    // For arealight
    float        width;       // radius for sphere light.
    float        height;
    float        u_dir[3];
    float        v_dir[3];

    // For sun
    float        turbidity;

    // For object light(ID to mesh object)
    int          mesh_id;

    // Light matrix
    float        transform[4][4];

    const char*  name;

} Light;

#endif // __LTE_LIGHT_H__
