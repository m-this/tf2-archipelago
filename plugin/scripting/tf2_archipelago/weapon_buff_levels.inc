/**
 * The weapon buff levels the run granted, and every decision that reads them.
 *
 * weapon_buffs.inc is the adapter: it resolves entities to weapon indexes,
 * reads hooked attribute values and entity properties, and writes what this
 * file decides onto the provider, the player or the victim. Nothing here
 * touches the engine, so gamedata/weapon_buff_levels_test.go runs it in
 * SourcePawn's standalone VM and the answers are measured rather than read.
 *
 * A function belongs here when it needs nothing but its arguments, this file's
 * state and the generated tables: no entity property, no SDKCall, no TF2
 * lookup, not even RoundToFloor or StrEqual, which the standalone VM does not
 * bind, and no constant out of a SourceMod include either. Amounts that the
 * engine wants as whole numbers come back as floats for the adapter to round.
 */

#define DamageEffect 0
#define FireRateEffect 1
#define ClipSizeEffect 3
#define ProjectileCountEffect 5
#define HealOnKillEffect 6
#define CritsOnKillEffect 7
#define BleedEffect 8
#define IgniteEffect 9
#define MadMilkEffect 14
#define GasolineEffect 15
#define MarkForDeathEffect 16
#define DestroyProjectilesEffect 19
#define HealOnHitEffect 21
#define SpeedOnKillEffect 22
#define MoveSpeedEffect 24
#define JumpHeightEffect 25
#define DisposableSentryEffect 36
#define MaxHealthEffect 38
#define UberOnHitEffect 39
#define JarateEffect 60
#define NoSelfBlastEffect 62
#define ActiveHealthRegenEffect 66
#define AmmoOnHitEffect 69
#define MinicritsOnKillEffect 70
#define RocketJumpProtectionEffect 72
#define RevealCloakedEffect 77
#define SpeedOnHitEffect 79
#define BaseHealthOnKillEffect 80

#define BuffMode_Add 1
#define BuffMode_Percentage 2
#define BuffMode_Toggle 3

// The multiplier a melee weapon's fire rate never goes below: +50% and no more.
#define MeleeFireRateFloor 0.5
// Measured on a server: critboost on kill lasts a second longer than the
// attribute says.
#define CritsOnKillExtraSeconds 1.0
// What a syringe lands instead of the saw's five percent. A syringe gun puts
// out hits far faster than a swing, so the same buff charges a fifth as much.
#define SyringeUberOnHitFraction 0.01
// What one level of rocket-jump protection takes off self-inflicted blast damage.
#define RocketJumpProtectionPerLevel 0.25
#define StompBaseFallDamageMultiplier 3.0
#define StompFallDamagePerLevel 5.0
#define GasPasserDefinition 1180
#define ConstructionPDADefinition 25
// Gas coats for ten seconds and the burn it starts lasts ten more.
#define GasCreditSeconds 25.0
#define ProjectileDestructionBaseCooldown 2.0
#define ProjectileDestructionCooldownStep 0.25
#define ProjectileDestructionMinimumCooldown 0.5

// MAXPLAYERS + 1 in SourceMod; tf2_archipelago.sp refuses to compile otherwise.
#define BuffLevelsClients 102

static int g_Levels[WeaponCount][WeaponEffectCount];

// --- Levels ---

void BuffLevels_Clear()
{
    for (int weapon = 0; weapon < WeaponCount; weapon++)
    {
        for (int effect = 0; effect < WeaponEffectCount; effect++)
        {
            g_Levels[weapon][effect] = 0;
        }
    }
}

// A toggle is on or off, so a second copy changes nothing; everything else adds.
void BuffLevels_Add(int weapon, int effect, int levels)
{
    g_Levels[weapon][effect] = g_WeaponEffectModes[effect] == BuffMode_Toggle
        ? 1 : g_Levels[weapon][effect] + levels;
}

// One received copy of the buff at index in g_WeaponBuffKeys.
void BuffLevels_Grant(int buff)
{
    BuffLevels_Add(g_WeaponBuffWeapon[buff], g_WeaponBuffEffect[buff], 1);
}

int BuffLevels_Of(int weapon, int effect)
{
    return weapon >= 0 ? g_Levels[weapon][effect] : 0;
}

// Every level on one weapon, all effects together.
int BuffLevels_Count(int weapon)
{
    int count;
    for (int effect = 0; effect < WeaponEffectCount; effect++)
    {
        count += g_Levels[weapon][effect];
    }
    return count;
}

int BuffLevels_Total()
{
    int total;
    for (int weapon = 0; weapon < WeaponCount; weapon++)
    {
        total += BuffLevels_Count(weapon);
    }
    return total;
}

// One effect summed over the weapons of a loadout, -1 for an empty slot.
// Levels add up: a Scout who rolled the same buff on two weapons carries both.
int BuffLevels_Loadout(const int[] weapons, int count, int effect)
{
    int levels;
    for (int slot = 0; slot < count; slot++)
    {
        levels += BuffLevels_Of(weapons[slot], effect);
    }
    return levels;
}

// --- Which weapon is which ---

int WeaponBuffs_ForDefinition(int definition)
{
    for (int index = 0; index < sizeof(g_WeaponByDefinition); index++)
    {
        if (g_WeaponByDefinition[index][0] == definition)
        {
            return g_WeaponByDefinition[index][1];
        }
    }
    return -1;
}

// StrEqual on a weapon's name, which the standalone VM does not bind.
static bool BuffLevels_Named(int weapon, const char[] name)
{
    if (weapon < 0)
    {
        return false;
    }
    for (int index = 0; g_WeaponNames[weapon][index] == name[index]; index++)
    {
        if (name[index] == '\0')
        {
            return true;
        }
    }
    return false;
}

enum BuffStomp
{
    BuffStomp_None,
    BuffStomp_Mantreads,
    BuffStomp_Thermal,
}

// The two items that hurt by landing. Their damage and kills reach the game
// as the player's, not the item's, so the plugin pays their buffs itself.
BuffStomp BuffLevels_Stomp(int weapon)
{
    if (BuffLevels_Named(weapon, "Mantreads"))
    {
        return BuffStomp_Mantreads;
    }
    if (BuffLevels_Named(weapon, "Thermal Thruster"))
    {
        return BuffStomp_Thermal;
    }
    return BuffStomp_None;
}

static bool BuffLevels_IsSyringeGun(int weapon)
{
    return BuffLevels_Named(weapon, "Blutsauger")
        || BuffLevels_Named(weapon, "Overdose")
        || BuffLevels_Named(weapon, "Syringe Gun");
}

// Only CTFMinigun reads attack_projectiles; the confirmed-shot sweep does the
// rest of the guns that can draw the buff.
static bool BuffLevels_NativeDestruction(int weapon)
{
    return BuffLevels_Named(weapon, "Brass Beast")
        || BuffLevels_Named(weapon, "Huo-Long Heater")
        || BuffLevels_Named(weapon, "Minigun")
        || BuffLevels_Named(weapon, "Natascha")
        || BuffLevels_Named(weapon, "Tomislav");
}

static bool BuffLevels_SweepsProjectiles(int weapon)
{
    static const char names[][] = {
        "Ambassador", "Baby Face's Blaster", "Back Scatter", "Bazaar Bargain", "Brass Beast",
        "Classic", "Cleaner's Carbine", "Diamondback", "Enforcer", "Family Business",
        "Force-a-Nature", "Frontier Justice", "Hitman's Heatmaker", "Huo-Long Heater",
        "L'Étranger", "Machina", "Minigun", "Natascha", "Panic Attack", "Pistol",
        "Pomson 6000", "Pretty Boy's Pocket Pistol", "Reserve Shooter", "Revolver",
        "Righteous Bison", "Scattergun", "Short Circuit", "Shortstop", "Shotgun", "SMG",
        "Sniper Rifle", "Soda Popper", "Sydney Sleeper", "Tomislav", "Widowmaker", "Winger"
    };
    for (int index = 0; index < sizeof(names); index++)
    {
        if (BuffLevels_Named(weapon, names[index]))
        {
            return true;
        }
    }
    return false;
}

// --- The provider ---

bool WeaponBuffs_IsPassiveEffect(int effect)
{
    return effect == MoveSpeedEffect || effect == JumpHeightEffect
        || effect == ActiveHealthRegenEffect || effect == MaxHealthEffect;
}

bool WeaponBuffs_IsOnKillEffect(int effect)
{
    return effect == HealOnKillEffect || effect == CritsOnKillEffect
        || effect == SpeedOnKillEffect || effect == MinicritsOnKillEffect
        || effect == BaseHealthOnKillEffect;
}

// Paid on a confirmed enemy hit. As provider attributes TF2 would also apply
// them to self-inflicted blast damage: a Rocket Jumper could bleed its owner.
static bool BuffLevels_IsManualHitEffect(int effect)
{
    return effect == BleedEffect || effect == IgniteEffect
        || effect == MadMilkEffect || effect == GasolineEffect
        || effect == MarkForDeathEffect || effect == JarateEffect;
}

/* The levels of effect to install on the provider while weapon is held, zero
 * when another path pays it. runBuffs is whether the holder may use the run's
 * buffs at all; innate is what his bot card grants on top.
 *
 * Passives follow the loadout (BuffLevels_PassiveValue) and health regeneration
 * is the timer's. Hit effects, ammo on hit, no-self-blast and the stomp items'
 * damage, on-kill and launch charge are paid by the plugin on their own events.
 * Disposable sentries are a Construction PDA effect predicted separately.
 */
int BuffLevels_OnProvider(int weapon, int effect, bool runBuffs, int innate)
{
    BuffStomp stomp = BuffLevels_Stomp(weapon);
    if (WeaponBuffs_IsPassiveEffect(effect) || BuffLevels_IsManualHitEffect(effect)
        || (effect == DestroyProjectilesEffect && !BuffLevels_NativeDestruction(weapon))
        || effect == AmmoOnHitEffect || effect == DisposableSentryEffect
        || effect == NoSelfBlastEffect
        || (stomp != BuffStomp_None
            && (effect == DamageEffect || WeaponBuffs_IsOnKillEffect(effect)))
        || (stomp == BuffStomp_Thermal && effect == ClipSizeEffect))
    {
        return 0;
    }
    return (runBuffs ? BuffLevels_Of(weapon, effect) : 0) + innate;
}

// Whether BuffLevels_ProviderValue divides by the hooked value of the effect's
// attribute class, which the adapter then has to read.
bool BuffLevels_ReadsBase(int effect)
{
    return g_WeaponEffectModes[effect] == BuffMode_Percentage;
}

/* What the provider's attribute is worth for levels of effect on weapon.
 *
 * A percentage composes with what MvM and the item already give: base is that
 * hooked value, and the provider carries the ratio that moves it by the buff.
 * Inverted percentages such as fire rate never reach zero, or the engine makes
 * attacks faster than the client channel carries them, and a melee swing past
 * +50% is faster than its own animation and lands nothing.
 */
float BuffLevels_ProviderValue(int weapon, int effect, int levels, float base, bool melee)
{
    if (g_WeaponEffectModes[effect] == BuffMode_Toggle)
    {
        return 1.0;
    }
    if (g_WeaponEffectModes[effect] == BuffMode_Add)
    {
        float increment = effect == UberOnHitEffect && BuffLevels_IsSyringeGun(weapon)
            ? SyringeUberOnHitFraction : g_WeaponEffectIncrements[effect];
        return increment * float(levels);
    }
    if (base < 0.0001 && base > -0.0001)
    {
        base = 1.0;
    }
    float combined = base + g_WeaponEffectIncrements[effect] * float(levels);
    float minimum = effect == FireRateEffect ? 0.2 : 0.05;
    if (effect == FireRateEffect && melee)
    {
        minimum = MeleeFireRateFloor;
    }
    if (combined < minimum)
    {
        combined = minimum;
    }
    return combined / base;
}

// A passive is a player attribute summed over the loadout, so no weapon and
// no melee floor: movement, jump height and max health.
float BuffLevels_PassiveValue(int effect, int levels, float base)
{
    return BuffLevels_ProviderValue(-1, effect, levels, base, false);
}

// --- Hits and kills ---

enum struct BuffReward
{
    float heal;
    float crits;
    float minicrits;
    bool speed;
}

// What a kill with weapon pays the killer. False when it pays nothing.
bool BuffLevels_KillReward(int weapon, BuffReward reward)
{
    int crits = BuffLevels_Of(weapon, CritsOnKillEffect);
    reward.heal = g_WeaponEffectIncrements[HealOnKillEffect]
        * float(BuffLevels_Of(weapon, HealOnKillEffect));
    reward.crits = crits > 0
        ? g_WeaponEffectIncrements[CritsOnKillEffect] * float(crits) + CritsOnKillExtraSeconds
        : 0.0;
    reward.minicrits = g_WeaponEffectIncrements[MinicritsOnKillEffect]
        * float(BuffLevels_Of(weapon, MinicritsOnKillEffect));
    reward.speed = BuffLevels_Of(weapon, SpeedOnKillEffect) > 0;
    return reward.heal > 0.0 || reward.crits > 0.0 || reward.minicrits > 0.0 || reward.speed;
}

// What a jar splash pays the thrower per soaked enemy. The game reads heal and
// speed on hit off a weapon attack, and a splash is not one.
bool BuffLevels_JarReward(int weapon, BuffReward reward)
{
    reward.heal = g_WeaponEffectIncrements[HealOnHitEffect]
        * float(BuffLevels_Of(weapon, HealOnHitEffect));
    reward.crits = 0.0;
    reward.minicrits = 0.0;
    reward.speed = BuffLevels_Of(weapon, SpeedOnHitEffect) > 0;
    return reward.heal > 0.0 || reward.speed;
}

/* Whether the game already paid the on-kill buffs of the weapon that killed.
 * It reads them through the killer's provider, which carries the weapon he
 * holds, so a kill by a holstered weapon (a burn after the Gas Passer was put
 * away) is the plugin's to pay.
 */
bool BuffLevels_GamePaidKill(int killedWith, int held)
{
    return killedWith < 0 || killedWith == held;
}

enum struct BuffHit
{
    bool reveal;
    float bleed;
    bool ignite;
    bool milk;
    bool gas;
    bool marked;
    float jarate;
}

// What a confirmed enemy hit with weapon does to the victim, durations in seconds.
bool BuffLevels_HitEffects(int weapon, BuffHit hit)
{
    int jarate = BuffLevels_Of(weapon, JarateEffect);
    hit.reveal = BuffLevels_Of(weapon, RevealCloakedEffect) > 0;
    hit.bleed = 3.0 * float(BuffLevels_Of(weapon, BleedEffect));
    hit.ignite = BuffLevels_Of(weapon, IgniteEffect) > 0;
    hit.milk = BuffLevels_Of(weapon, MadMilkEffect) > 0;
    hit.gas = BuffLevels_Of(weapon, GasolineEffect) > 0;
    hit.marked = BuffLevels_Of(weapon, MarkForDeathEffect) > 0;
    hit.jarate = jarate > 0 ? 10.0 + 2.0 * float(jarate) : 0.0;
    return hit.reveal || hit.bleed > 0.0 || hit.ignite || hit.milk || hit.gas
        || hit.marked || hit.jarate > 0.0;
}

// Who gassed each victim, and when. A burn death within GasCreditSeconds is
// the Gas Passer's kill too, whichever weapon lit the gas.
static int g_GasFrom[BuffLevelsClients];
static float g_GasAt[BuffLevelsClients];

void BuffLevels_Gassed(int victim, int thrower, float now)
{
    g_GasFrom[victim] = thrower;
    g_GasAt[victim] = now;
}

// The Gas Passer, when this burn death owes it its on-kill buffs; -1 when the
// game credited it already or the gas was someone else's or too old. Pays once.
int BuffLevels_GasKill(int victim, int attacker, int killedWith, float now)
{
    int gasPasser = WeaponBuffs_ForDefinition(GasPasserDefinition);
    if (gasPasser < 0 || killedWith == gasPasser || g_GasFrom[victim] != attacker
        || now - g_GasAt[victim] > GasCreditSeconds)
    {
        return -1;
    }
    g_GasFrom[victim] = 0;
    return gasPasser;
}

// --- Stomps ---

// What the stomp's fall damage is multiplied by. 1.0 changes nothing.
float BuffLevels_StompDamageScale(int weapon)
{
    int levels = BuffLevels_Of(weapon, DamageEffect);
    if (BuffLevels_Stomp(weapon) == BuffStomp_None || levels <= 0)
    {
        return 1.0;
    }
    return (StompBaseFallDamageMultiplier + StompFallDamagePerLevel * float(levels))
        / StompBaseFallDamageMultiplier;
}

// Health a lethal stomp gives back, as a share of the class's base health.
// The adapter rounds it up and caps it at the overheal limit.
float BuffLevels_StompKillHeal(int weapon, int classHealth)
{
    int levels = BuffLevels_Of(weapon, BaseHealthOnKillEffect);
    if (levels <= 0 || classHealth <= 0)
    {
        return 0.0;
    }
    return float(classHealth) * g_WeaponEffectIncrements[BaseHealthOnKillEffect]
        * float(levels) / 100.0;
}

// --- Self-inflicted blasts ---

// The health each player had before the first blast of his own this frame,
// zero when none is pending.
static int g_SelfBlastHealth[BuffLevelsClients];

int WeaponBuffs_SelfBlastBaseline(int pending, int health)
{
    return pending > 0 ? pending : health;
}

float WeaponBuffs_SelfBlastGuardHealth(int health, float damage)
{
    return damage >= float(health) ? float(health) + damage : float(health);
}

// The share of self-inflicted blast damage rocket-jump protection keeps.
float BuffLevels_SelfBlastKept(int weapon)
{
    int levels = BuffLevels_Of(weapon, RocketJumpProtectionEffect);
    if (levels <= 0)
    {
        return 1.0;
    }
    float kept = 1.0 - RocketJumpProtectionPerLevel * float(levels);
    return kept > 0.0 ? kept : 0.0;
}

/* No-self-blast lets the full damage through, so the blast pushes as it should,
 * and gives the health back after. victim was at health when his own blast hit.
 * True on the first hit since the last refund: the adapter books the refund
 * then, because uber and other rules can skip the alive-post hook.
 */
bool BuffLevels_SelfBlastHit(int victim, int health)
{
    bool first = g_SelfBlastHealth[victim] == 0;
    g_SelfBlastHealth[victim] = WeaponBuffs_SelfBlastBaseline(g_SelfBlastHealth[victim], health);
    return first;
}

// The health to give back, zero when nothing is pending. Pays once.
int BuffLevels_SelfBlastRefund(int victim)
{
    int health = g_SelfBlastHealth[victim];
    g_SelfBlastHealth[victim] = 0;
    return health;
}

// --- Disposable sentries ---

// Both PDAs share the Construction PDA's pool.
int BuffLevels_DisposableSentries()
{
    return BuffLevels_Of(WeaponBuffs_ForDefinition(ConstructionPDADefinition),
        DisposableSentryEffect);
}

// Whether an Engineer at the build limit may place one more disposable sentry:
// he needs his primary sentry, and the allowance is the run's levels on top of
// what the upgrade station sold him.
bool BuffLevels_DisposableRoom(int sold, bool hasPrimary, int disposable)
{
    int levels = BuffLevels_DisposableSentries();
    return levels > 0 && hasPrimary && disposable < levels + sold;
}

/* The disposable-sentry level the client must predict while holding the item
 * at definition, zero for none. Placing from the toolbox takes one slot, so it
 * shows one more. At the upgrade station the shop must see only what it sold.
 */
int BuffLevels_DisposablePrediction(int definition, bool atStation)
{
    int levels = BuffLevels_DisposableSentries();
    if (atStation || levels <= 0)
    {
        return 0;
    }
    if (definition == ConstructionPDADefinition || definition == 737 || definition == 15025)
    {
        return levels;
    }
    if (definition == 28 || definition == 15028)
    {
        return levels + 1;
    }
    return 0;
}

// --- Projectile destruction on guns ---

float WeaponBuffs_ProjectileDestructionCooldown(int levels)
{
    float cooldown = ProjectileDestructionBaseCooldown
        - float(levels - 1) * ProjectileDestructionCooldownStep;
    return cooldown > ProjectileDestructionMinimumCooldown
        ? cooldown : ProjectileDestructionMinimumCooldown;
}

// The last state seen of each player's held weapon. -1 for a clip or ammo the
// weapon does not have, below zero for no energy meter.
static int g_ShotReference[BuffLevelsClients];
static int g_ShotClip[BuffLevelsClients];
static int g_ShotAmmo[BuffLevelsClients];
static float g_ShotEnergy[BuffLevelsClients];
static float g_ShotNextAttack[BuffLevelsClients];

void BuffLevels_ForgetShot(int client)
{
    g_ShotReference[client] = 0;
}

/* One think of client holding weapon (entity reference, catalogue index).
 * Returns the destruction levels to sweep with when a shot was confirmed since
 * the last think, zero otherwise.
 *
 * A shot is ammo, clip or energy spent, or the next attack pushed back while
 * there was something to fire. Comparing against the same entity only: a weapon
 * switch is not a shot. Miniguns are the game's own (BuffLevels_OnProvider).
 */
int BuffLevels_Shot(int client, int weapon, int reference, int clip, int ammo,
    float energy, float nextAttack)
{
    int levels;
    if (g_ShotReference[client] == reference)
    {
        bool spent = (clip >= 0 && clip < g_ShotClip[client])
            || (ammo >= 0 && ammo < g_ShotAmmo[client])
            || (energy >= 0.0 && energy + 0.001 < g_ShotEnergy[client]);
        bool loaded = clip >= 0 ? g_ShotClip[client] > 0
            : (energy >= 0.0 ? g_ShotEnergy[client] > 0.0 : g_ShotAmmo[client] > 0);
        bool fired = spent || (loaded && nextAttack > g_ShotNextAttack[client] + 0.001);
        if (fired && BuffLevels_SweepsProjectiles(weapon) && !BuffLevels_NativeDestruction(weapon))
        {
            levels = BuffLevels_Of(weapon, DestroyProjectilesEffect);
        }
    }
    g_ShotReference[client] = reference;
    g_ShotClip[client] = clip;
    g_ShotAmmo[client] = ammo;
    g_ShotEnergy[client] = energy;
    g_ShotNextAttack[client] = nextAttack;
    return levels;
}

// A new occupant of the client index starts with nothing pending.
void BuffLevels_ForgetClient(int client)
{
    BuffLevels_ForgetShot(client);
    g_SelfBlastHealth[client] = 0;
}

// A buff rebuild must not leave more ammo than it found: on Windows, setting a
// percentage max-ammo attribute scales the held ammo again on every rebuild. A
// spawn or a resupply is the exception, as the game means the player to be full,
// so there the ceiling is the buffed maximum. A negative maximum is no refill.
int BuffLevels_AmmoKept(int saved, int now, int maximum)
{
    int ceiling = saved;
    if (maximum > ceiling)
    {
        ceiling = maximum;
    }
    if (now > ceiling)
    {
        return ceiling;
    }
    return now;
}
