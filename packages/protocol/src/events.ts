import { z } from "zod";

export const eventEnvelopeSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("event.commandAccepted"),
    commandId: z.string(),
    machineId: z.string(),
  }),
  z.object({
    type: z.literal("event.turn.delta"),
    threadId: z.string(),
    turnId: z.string(),
    delta: z.string(),
  }),
  z.object({
    type: z.literal("event.resource.changed"),
    machineId: z.string(),
    resourceId: z.string(),
  }),
]);
