import { cn } from "@/utils";
import { hashColor, initials } from "@/utils/color-hash";

interface PersonAvatarProps {
  name: string;
  /** Extra text (e.g. email) folded into the color hash so two different
   *  people who happen to share a display name still get different colors. */
  seed?: string;
  size?: "xs" | "sm" | "md" | "lg";
  className?: string;
}

const SIZE_CLASSES: Record<NonNullable<PersonAvatarProps["size"]>, string> = {
  xs: "size-7 text-[11px]",
  sm: "size-9 text-xs",
  md: "size-10 text-sm",
  lg: "size-11 text-base",
};

export function PersonAvatar({ name, seed, size = "sm", className }: PersonAvatarProps) {
  return (
    <div
      title={name}
      className={cn(
        "flex shrink-0 items-center justify-center rounded-full font-semibold text-white select-none",
        SIZE_CLASSES[size],
        className
      )}
      style={{ backgroundColor: hashColor(seed ?? name) }}
    >
      {initials(name)}
    </div>
  );
}
