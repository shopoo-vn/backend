import { PartialType } from '@nestjs/mapped-types';
import { CreateListingDto } from './create-listing.dto';

// All fields optional. Status is NOT user-editable — it changes only via
// moderation events (listing.approved / listing.rejected).
export class UpdateListingDto extends PartialType(CreateListingDto) {}
