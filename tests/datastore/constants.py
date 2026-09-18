
from enum import IntEnum, IntFlag

# * https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Constants
# * https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Enums
# * https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Flags

MAX_PERIOD = 365
"""The max value an object validity period can be, in days"""

MAX_METABIN_SIZE = 1024
"""The max length an object MetaBinary can be"""

DATASTOREPERMISSION_RECIPIENTIDS_MAX = 100
"""The max number of recipients that can be assigned in a `DataStorePermission`"""

INVALID_DATAID = 0
"""
Invalid object DataID. This is used when either in "zeroed" data or when a
value is not needed
"""

INVALID_DATA_TYPE = 65535
"""
Invalid object data type. This is used when either in "zeroed" data or when a
value is not needed
"""

INVALID_PASSWORD = 0
"""
Invalid object password. This is used when either in "zeroed" data or when a
value is not needed
"""

MAX_NAME_LENGTH = 64
"""The max length an object name can be"""

MAX_SEARCH_RESULT_SIZE = 100
"""The max number of search results that can be returned when not using `RESULTRANGE_ANY_OFFSET`"""

MAX_SEARCH_ANY_RESULT_SIZE = 20
"""The max number of search results that can be returned when using `RESULTRANGE_ANY_OFFSET`"""

MAX_SEARCH_DATA_TYPE_SIZE = 10
"""The max number of object data types that can be set in `DataStoreSearchParam`"""

NUM_TAG_SLOT = 16
"""The max number of tags an object may have"""

RATING_SLOT_MAX = 15
"""The max rating slot index (0-based)"""

NUM_RATING_SLOT = 16
"""The max number of rating slots an object may have"""

MAX_TAG_LENGTH = 24
"""The max length an object tag can be"""

DEFAULT_PERIOD = 90
"""The default object validity period, in days"""

DEFAULT_HTTP_THREAD_PRIORITY = 16
"""Unknown"""

DEFAULT_RELAY_BUFFER_SIZE = 16384
"""Unknown"""

DEFAULT_HTTP_BUFFER_SIZE = 32768
"""Unknown"""

DEFAULT_DATA_TRANSFER_TIMEOUT_BYTES_PER_SECOND = 167
"""Unknown"""

DEFAULT_DATA_TRANSFER_MINIMUM_TIMEOUT = 60000
"""Unknown"""

DEFAULT_HTTP_SEND_SOCKET_BUFFER_SIZE = 65536
"""Unknown"""

DEFAULT_HTTP_RECV_SOCKET_BUFFER_SIZE = 65536
"""Unknown"""

INVALID_PERSISTENCE_SLOT_ID = 65535
"""
Invalid object persistence storage slot ID. This is used when either in
"zeroed" data or when a value is not needed
"""

NUM_PERSISTENCE_SLOT = 16
"""The max number of objects each user can have in object persistence storage"""

BATCH_PROCESSING_CAPACITY_POST_OBJECT = 16
"""
The max number of objects allowed in "posting batch" methods (such as
`RateObjectsWithPosting`) in a single RMC
"""

BATCH_PROCESSING_CAPACITY = 100
"""
The max number of objects allowed in non-posting "batch" methods (such as
`GetMetas`) in a single RMC
"""

RESULTRANGE_ANY_OFFSET = 4294967295
"""If used on a `ResultRange`, the server picks a random offset"""

RESULTRANGE_DEFAULT_SIZE = 20
"""The default number of entries a `ResultRange` can return"""

class Permission(IntEnum):
	"""Determines the permission level of an object in `DataStorePermission`"""

	PERMISSION_PUBLIC = 0
	"""Anyone can perform the operation"""

	PERMISSION_FRIEND = 1
	"""Only the object owner and their friends can perform the operation"""

	PERMISSION_SPECIFIED = 2
	"""Only the object owner and the specified users can perform the operation"""

	PERMISSION_PRIVATE = 3
	"""Only the object owner can perform the operation"""

	PERMISSION_SPECIFIED_FRIEND = 4
	"""
	Only the object owner and the specified friends can perform the operation.
	If a friend is specified and then removed as a friend, that user loses
	permission
	"""

class DataStatus(IntEnum):
	"""Determines the availability status of an object"""

	DATA_STATUS_NONE = 0
	"""
	The object has been successfully uploaded and is available to anyone with
	access permissions. See `DataStorePermission`
	"""

	DATA_STATUS_PENDING = 2
	"""
	The object is uploaded but pending review. This state is automatically set
	when an object is uploaded with `DATA_FLAG_NEED_REVIEW` set. The object can
	only be seen by the owner until review is finished. Anyone besides the
	owner who tries to query for objects under review will result in
	`DataStore::UnderReviewing`
	"""

	DATA_STATUS_REJECTED = 5
	"""
	The object has failed review and has been rejected. The object is not
	available to anyone, including the owner, and behaves as if the object does
	not exist (`DataStore::NotFound`). The only exception to this is if the
	owner uses `SearchObject` or `SearchObjectLight` with either
	`SEARCH_TYPE_OWN_REJECTED` or `SEARCH_TYPE_OWN_ALL`
	"""

class SearchType(IntEnum):
	"""Determines the type of objects allowed to be returned when using `DataStoreSearchParam`"""

	SEARCH_TYPE_PUBLIC = 1
	"""Objects who have their access permission set to `PERMISSION_PUBLIC`"""

	SEARCH_TYPE_SEND_FRIEND = 2
	"""
	Objects owned by the current user and which have their access permission
	set to `PERMISSION_FRIEND`
	"""

	SEARCH_TYPE_SEND_SPECIFIED = 3
	"""
	Objects owned by the current user and which have their access permission
	set to `PERMISSION_SPECIFIED`
	"""

	SEARCH_TYPE_SEND_SPECIFIED_FRIEND = 4
	"""
	Objects owned by the current user and which have their access permission
	set to `PERMISSION_SPECIFIED_FRIEND`
	"""

	SEARCH_TYPE_SEND = 5
	"""
	Objects owned by the current user and which have their access permission
	set to `PERMISSION_FRIEND`, `PERMISSION_SPECIFIED` or
	`PERMISSION_SPECIFIED_FRIEND`
	"""

	SEARCH_TYPE_FRIEND = 6
	"""
	Objects owned by the friends of the current user which have their access
	permission set to `PERMISSION_FRIEND`
	"""

	SEARCH_TYPE_RECEIVED_SPECIFIED = 7
	"""
	Objects which have their access permission set to either
	`PERMISSION_SPECIFIED` or `PERMISSION_SPECIFIED_FRIEND` and which have the
	current user set as a recipient
	"""

	SEARCH_TYPE_RECEIVED = 8
	"""
	Objects which have their access permission set to either
	`PERMISSION_SPECIFIED` or `PERMISSION_SPECIFIED_FRIEND` and which have the
	current user set as a recipient, or objects owned by the friends of the
	current user which have their access permission set to `PERMISSION_FRIEND`
	"""

	SEARCH_TYPE_PRIVATE = 9
	"""
	Objects owned by the current user and which have their access permission
	set to `PERMISSION_PRIVATE`
	"""

	SEARCH_TYPE_OWN = 10
	"""
	Objects owned by the current user and which have their data status set to
	`DATA_STATUS_NONE`
	"""

	SEARCH_TYPE_PUBLIC_EXCLUDE_OWN_AND_FRIENDS = 11
	"""
	Objects who have their access permission set to `PERMISSION_PUBLIC`,
	excluding those uploaded by the current user and their friends
	"""

	SEARCH_TYPE_OWN_PENDING = 12
	"""
	Objects owned by the current user and which have their data status set to
	`DATA_STATUS_PENDING`
	"""

	SEARCH_TYPE_OWN_REJECTED = 13
	"""
	Objects owned by the current user and which have their data status set to
	`DATA_STATUS_REJECTED`
	"""

	SEARCH_TYPE_OWN_ALL = 14
	"""
	Objects owned by the current user and which have their data status set to
	`DATA_STATUS_NONE`, `DATA_STATUS_PENDING` or `DATA_STATUS_REJECTED`
	"""

class SearchTarget(IntEnum):
	"""Determines the target object owner type when using `DataStoreSearchParam`"""

	SEARCH_TARGET_ANYBODY = 0
	"""Objects owned by anyone"""

	SEARCH_TARGET_FRIEND = 1
	"""Objects owned by the friends of the current user"""

	SEARCH_TARGET_ANYBODY_EXCLUDE_SPECIFIED = 2
	"""Exclude objects owned by the users specified in the search paramaters"""

class SearchSortColumn(IntEnum):
	"""Determines the database column to sort search results by"""

	SEARCH_SORT_COLUMN_DATAID = 0
	"""Sort by object DataID"""

	SEARCH_SORT_COLUMN_SIZE = 1
	"""Sort by object size"""

	SEARCH_SORT_COLUMN_NAME_ALPHABETICAL = 2
	"""Sort by object name, alphabetical"""

	SEARCH_SORT_COLUMN_DATA_TYPE = 3
	"""Sort by object data type"""

	SEARCH_SORT_COLUMN_REFERRED_COUNT = 4
	"""Sort by object reference count"""

	SEARCH_SORT_COLUMN_CREATED_TIME = 5
	"""Sort by object creation time"""

	SEARCH_SORT_COLUMN_UPDATED_TIME = 6
	"""Sort by object modified time"""

	SEARCH_SORT_COLUMN_RATING0 = 64
	"""Sort by the total combined ratings in rating slot 0"""

	SEARCH_SORT_COLUMN_RATING1 = 65
	"""Sort by the total combined ratings in rating slot 1"""

	SEARCH_SORT_COLUMN_RATING2 = 66
	"""Sort by the total combined ratings in rating slot 2"""

	SEARCH_SORT_COLUMN_RATING3 = 67
	"""Sort by the total combined ratings in rating slot 3"""

	SEARCH_SORT_COLUMN_RATING4 = 68
	"""Sort by the total combined ratings in rating slot 4"""

	SEARCH_SORT_COLUMN_RATING5 = 69
	"""Sort by the total combined ratings in rating slot 5"""

	SEARCH_SORT_COLUMN_RATING6 = 70
	"""Sort by the total combined ratings in rating slot 6"""

	SEARCH_SORT_COLUMN_RATING7 = 71
	"""Sort by the total combined ratings in rating slot 7"""

	SEARCH_SORT_COLUMN_RATING8 = 72
	"""Sort by the total combined ratings in rating slot 8"""

	SEARCH_SORT_COLUMN_RATING9 = 73
	"""Sort by the total combined ratings in rating slot 9"""

	SEARCH_SORT_COLUMN_RATING10 = 74
	"""Sort by the total combined ratings in rating slot 10"""

	SEARCH_SORT_COLUMN_RATING11 = 75
	"""Sort by the total combined ratings in rating slot 11"""

	SEARCH_SORT_COLUMN_RATING12 = 76
	"""Sort by the total combined ratings in rating slot 12"""

	SEARCH_SORT_COLUMN_RATING13 = 77
	"""Sort by the total combined ratings in rating slot 13"""

	SEARCH_SORT_COLUMN_RATING14 = 78
	"""Sort by the total combined ratings in rating slot 14"""

	SEARCH_SORT_COLUMN_RATING15 = 79
	"""Sort by the total combined ratings in rating slot 15"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE0 = 96
	"""Sort by the average of all ratings in rating slot 0"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE1 = 97
	"""Sort by the average of all ratings in rating slot 1"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE2 = 98
	"""Sort by the average of all ratings in rating slot 2"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE3 = 99
	"""Sort by the average of all ratings in rating slot 3"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE4 = 100
	"""Sort by the average of all ratings in rating slot 4"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE5 = 101
	"""Sort by the average of all ratings in rating slot 5"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE6 = 102
	"""Sort by the average of all ratings in rating slot 6"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE7 = 103
	"""Sort by the average of all ratings in rating slot 7"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE8 = 104
	"""Sort by the average of all ratings in rating slot 8"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE9 = 105
	"""Sort by the average of all ratings in rating slot 9"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE10 = 106
	"""Sort by the average of all ratings in rating slot 10"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE11 = 107
	"""Sort by the average of all ratings in rating slot 11"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE12 = 108
	"""Sort by the average of all ratings in rating slot 12"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE13 = 109
	"""Sort by the average of all ratings in rating slot 13"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE14 = 110
	"""Sort by the average of all ratings in rating slot 14"""

	SEARCH_SORT_COLUMN_RATING_AVERAGE15 = 111
	"""Sort by the average of all ratings in rating slot 15"""

class SearchSortOrder(IntEnum):
	"""Determines the direction to order search results"""

	SEARCH_SORT_ORDER_ASC = 0
	"""Ascending order"""

	SEARCH_SORT_ORDER_DESC = 1
	"""Descending order"""

class RatingLockType(IntEnum):
	"""Determines how to lock the current user out of future ratings on the slot"""

	RATING_LOCK_NONE = 0
	"""No locking used. The user is allowed to freely rate the slot as much as they want"""

	RATING_LOCK_INTERVAL = 1
	"""The user cannot rate the same slot again for a specified amount of time"""

	RATING_LOCK_PERIOD = 2
	"""The user cannot rate the same slot again until a specific date/time"""

	RATING_LOCK_PERMANENT = 3
	"""The user can never rate the slot again"""

class RatingLockPeriod(IntEnum):
	"""Determines the specific day a rating lock is removed when using `RATING_LOCK_PERIOD`"""

	RATING_LOCK_PERIOD_DAY1 = -17
	"""Unlocks on the first day of the following month"""

	RATING_LOCK_PERIOD_SUN = -7
	"""Unlocks on the Sunday of the following week"""

	RATING_LOCK_PERIOD_SAT = -6
	"""Unlocks on the Saturday of the following week"""

	RATING_LOCK_PERIOD_FRI = -5
	"""Unlocks on the Friday of the following week"""

	RATING_LOCK_PERIOD_THU = -4
	"""Unlocks on the Thursday of the following week"""

	RATING_LOCK_PERIOD_WED = -3
	"""Unlocks on the Wednesday of the following week"""

	RATING_LOCK_PERIOD_TUE = -2
	"""Unlocks on the Tuesday of the following week"""

	RATING_LOCK_PERIOD_MON = -1
	"""Unlocks on the Monday of the following week"""

class SearchResultTotalCountType(IntEnum):
	"""Tells the client how to interpret the result count on `DataStoreSearchResult`"""

	SEARCH_RESULT_TOTAL_EXACT = 0
	"""The search result count is the exact number of objects matching the criteria"""

	SEARCH_RESULT_TOTAL_MINIMUM = 1
	"""
	The search result count is the minimum number of objects matching the
	criteria. There are more matches beyond the returned results
	"""

	SEARCH_RESULT_TOTAL_ESTIMATE = 2
	"""The search result count is an estimate of the number of objects matching the criteria"""

	SEARCH_RESULT_TOTAL_DISABLED = 3
	"""The search result count is not populated"""

class DataFlag(IntFlag):
	"""General object flags"""

	DATA_FLAG_NONE = 0x00
	"""No flags set"""

	DATA_FLAG_NEED_REVIEW = 0x01
	"""The object needs to be reviewed before it becomes available. See `DataStatus`"""

	DATA_FLAG_PERIOD_FROM_LAST_REFERRED = 0x02
	"""The object can have it's validity period extended when referenced"""

	DATA_FLAG_USE_READ_LOCK = 0x04
	"""Enables object read locking. See the object lock ID"""

	DATA_FLAG_USE_NOTIFICATION_ON_POST = 0x08
	"""
	Creates an object notification for the object recipients when the object
	has finished uploading. This cannot be used with objects that do not use
	the external file server
	"""

	DATA_FLAG_USE_NOTIFICATION_ON_UPDATE = 0x10
	"""
	Creates an object notification for the object recipients when the object
	has finished updating. This cannot be used with objects that do not use the
	external file server
	"""

	DATA_FLAG_NOT_USE_FILESERVER = 0x20
	"""
	The object does not use the external file server and only exists as an
	object MetaBinary on the game server. This flag can NOT be set by the
	caller, it is automatically assigned to objects by the server. Attempting
	to set this flag manually will result in `DataStore::InvalidArgument`
	"""

	DATA_FLAG_NEED_COMPLETION = 0x40
	"""
	The object must have "completion" called when uploading/updating. Objects
	with this flag set will return `DataStore::NotFound` when queried for until
	upload/update is completed. Objects which have not been marked as
	"completed" will be deleted a few hours after upload, and the owner will no
	longer be able to mark them as "completed". In practice this flag seems to
	be ignored, and ALL objects (regardless of this flag) require "completion"
	to be called?
	"""

class ModificationFlag(IntFlag):
	"""Used to indicate what properties of an object have been modified"""

	MODIFICATION_FLAG_NONE = 0x000
	"""No changes"""

	MODIFICATION_FLAG_NAME = 0x001
	"""The object name has been updated"""

	MODIFICATION_FLAG_ACCESS_PERMISSION = 0x002
	"""The object access permission has been updated"""

	MODIFICATION_FLAG_UPDATE_PERMISSION = 0x004
	"""The object update permission (also seen as "delete permission") has been updated"""

	MODIFICATION_FLAG_PERIOD = 0x008
	"""The object validity period has been updated"""

	MODIFICATION_FLAG_METABINARY = 0x010
	"""The object MetaBinary has been updated"""

	MODIFICATION_FLAG_TAGS = 0x020
	"""The object tags have been updated"""

	MODIFICATION_FLAG_UPDATED_TIME = 0x040
	"""The object modified time has been updated"""

	MODIFICATION_FLAG_DATA_TYPE = 0x080
	"""The object data type has been updated"""

	MODIFICATION_FLAG_REFERRED_COUNT = 0x100
	"""The object reference count has been updated"""

	MODIFICATION_FLAG_STATUS = 0x200
	"""The object status has been updated"""

class ComparisonFlag(IntFlag):
	"""
	Before the objects data is updated, the current object data is checked
	against the data in `DataStoreChangeMetaCompareParam`. These flags tell the
	server which data to compare. If any of the compared fields do not match,
	the object data is not updated and `DataStore::ValueNotEqual` is thrown
	"""

	COMPARISON_FLAG_NONE = 0x000
	"""No comparison"""

	COMPARISON_FLAG_NAME = 0x001
	"""Compare names"""

	COMPARISON_FLAG_ACCESS_PERMISSION = 0x002
	"""Compare access permissions"""

	COMPARISON_FLAG_UPDATE_PERMISSION = 0x004
	"""Compare update permissions"""

	COMPARISON_FLAG_PERIOD = 0x008
	"""Compare periods"""

	COMPARISON_FLAG_METABINARY = 0x010
	"""Compare MetaBinaries"""

	COMPARISON_FLAG_TAGS = 0x020
	"""Compare tags"""

	COMPARISON_FLAG_DATA_TYPE = 0x040
	"""Compare data types"""

	COMPARISON_FLAG_REFERRED_COUNT = 0x080
	"""Compare reference counts"""

	COMPARISON_FLAG_STATUS = 0x100
	"""Compare statuses"""

	COMPARISON_FLAG_ALL = 0x200
	"""Compare everything"""

class ResultFlag(IntFlag):
	"""
	Determines what fields should be populated in object metadata. Fields not
	mentioned here are always populated
	"""

	RESULT_FLAG_TAGS = 0x1
	"""Return object tags"""

	RESULT_FLAG_RATINGS = 0x2
	"""Return object ratings"""

	RESULT_FLAG_METABINARY = 0x4
	"""Return the object MetaBinary"""

	RESULT_FLAG_PERMITTED_IDS = 0x8
	"""Return the recipient ID lists from both permissions"""

class RatingFlag(IntFlag):
	"""Determines how to handle how user ratings are applied to objects"""

	RATING_FLAG_MODIFIABLE = 0x04
	"""
	When set, each user can only have one rating per slot. If the user has
	already rated the slot, update the existing rating rather than create
	another rating entry
	"""

	RATING_FLAG_ROUND_MINUS = 0x08
	"""When set, ratings less than 0 are rounded to 0"""

	RATING_FLAG_DISABLE_SELF_RATING = 0x10
	"""
	When set, disables the ability for the object owner to rate the object. If
	the owner tries to rate the object, `DataStore::OperationNotAllowed` is
	thrown
	"""

class RatingInternalFlag(IntFlag):
	"""Determines how a rating slot should handle an incoming rating value"""

	RATING_INTERNAL_FLAG_USE_RANGE_MIN = 0x2
	"""
	When set, the incoming rating is checked against the minimum configured
	value. If below the minimum, `DataStore::InvalidArgument` is thrown. If not
	set, do not check for a minimum value
	"""

	RATING_INTERNAL_FLAG_USE_RANGE_MAX = 0x4
	"""
	When set, the incoming rating is checked against the maximum configured
	value. If above the maximum, `DataStore::InvalidArgument` is thrown. If not
	set, do not check for a maximum value
	"""
